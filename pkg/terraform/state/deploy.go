package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/digest"
	"github.com/sirupsen/logrus"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Input struct {
	Configs    []*coreV1.ConfigMap
	EnvVars    []coreV1.EnvVar
	Executions map[string]string
	Image      string
	Module     *v1.Module
	Secrets    []*coreV1.Secret
}

// deployCreate creates all resources for the job to run terraform create and returns the run name
func (h *handler) deployCreate(state *v1.State, input *Input) (*v1.Execution, error) {
	runHash := createRunHash(state, input, ActionCreate)
	jsonVars, err := json.Marshal(getCombinedVars(state, input))
	if err != nil {
		return &v1.Execution{}, err
	}

	namespace := state.Namespace
	or := []metaV1.OwnerReference{
		{
			APIVersion: "terraformcontroller.cattle.io/v1",
			Kind:       "State",
			Name:       state.Name,
			UID:        state.UID,
		},
	}

	logrus.Debug("Create - Creating execution")
	//skip owner reference for executions so logs stay around after deletion
	exec, err := h.createExecution([]metaV1.OwnerReference{}, state, input, runHash)
	if err != nil {
		return exec, err
	}

	logrus.Debug("Create - Creating secret")
	secret, err := h.createSecretForVariablesFile(or, exec.Name, state, jsonVars)
	if err != nil {
		return exec, err
	}

	logrus.Debug("Create - Creating serviceAccount")
	sa, err := h.createServiceAccount(exec.Name, namespace)
	if err != nil {
		return exec, err
	}

	logrus.Debug("Create - Creating clusterRoleBinding")
	rb, err := h.createClusterRoleBinding(or, exec.Name, "cluster-admin", sa.Name, namespace)
	if err != nil {
		return exec, err
	}

	logrus.Debug("Create - Creating job")
	job, err := h.createJob(or, input, exec.Name, runHash, ActionCreate, sa.Name, namespace)
	if err != nil {
		return exec, err
	}

	logrus.Debug("CreateUpdating owner references")
	err = h.updateOwnerReference(job, []interface{}{sa, rb, secret}, namespace)
	if err != nil {
		return exec, err
	}

	logrus.Infof("Deployed create job for state %v", state.Name)
	return exec, nil
}

// deployCreate creates all resources for the job to run terraform destroy
func (h *handler) deployDestroy(state *v1.State, input *Input) (*v1.Execution, error) {
	runHash := createRunHash(state, input, ActionDestroy)
	jsonVars, err := json.Marshal(getCombinedVars(state, input))
	if err != nil {
		return nil, err
	}

	or := []metaV1.OwnerReference{}

	logrus.Debug("Destroy - Creating execution")
	exec, err := h.createExecution(or, state, input, runHash)
	if err != nil {
		return exec, err
	}

	logrus.Debug("Destroy - Creating secret")
	secret, err := h.createSecretForVariablesFile(or, exec.Name, state, jsonVars)
	if err != nil {
		return exec, err
	}

	logrus.Debug("Destroy - Creating serviceAccount")
	sa, err := h.createServiceAccount(exec.Name, state.Namespace)
	if err != nil {
		return exec, err
	}

	logrus.Debug("Destroy - Creating clusterRoleBinding")
	rb, err := h.createClusterRoleBinding(or, exec.Name, "cluster-admin", sa.Name, state.Namespace)
	if err != nil {
		return exec, err
	}

	logrus.Debug("Destroy - Creating job")
	job, err := h.createJob(or, input, exec.Name, runHash, ActionDestroy, sa.Name, state.Namespace)
	if err != nil {
		return exec, err
	}

	logrus.Debug("Destroy - Updating owner references")
	err = h.updateOwnerReference(job, []interface{}{sa, rb, secret}, state.Namespace)
	if err != nil {
		return exec, err
	}

	logrus.Infof("Deployed destroy job for state %v with execution name %s", state.Name, exec.Name)
	return exec, nil
}

func (h *handler) createExecution(
	or []metaV1.OwnerReference,
	state *v1.State,
	input *Input,
	runHash string,
) (*v1.Execution, error) {
	execution := &v1.Execution{
		ObjectMeta: metaV1.ObjectMeta{
			GenerateName:    state.Name + "-",
			Namespace:       state.Namespace,
			OwnerReferences: or,
			Annotations:     map[string]string{"approved": ""},
			Labels: map[string]string{
				"state":   state.Name,
				"runHash": runHash,
			},
		},
		Spec: v1.ExecutionSpec{
			ExecutionName:    state.Name,
			AutoConfirm:      state.Spec.AutoConfirm,
			Content:          input.Module.Status.Content,
			ContentHash:      input.Module.Status.ContentHash,
			RunHash:          runHash,
			ExecutionVersion: state.Spec.Version,
		},
	}

	exec, err := h.executions.Create(execution)
	if err != nil {
		return nil, err
	}

	exec.Spec.SecretName = "s-" + exec.Name
	return h.executions.Update(exec)
}

func (h *handler) createSecretForVariablesFile(or []metaV1.OwnerReference, name string, execution *v1.State, vars []byte) (*coreV1.Secret, error) {
	secretData := map[string][]byte{
		"varFile": vars,
	}

	secret := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            "s-" + name,
			Namespace:       execution.Namespace,
			OwnerReferences: or,
		},
		Data: secretData,
	}

	s, err := h.secrets.Create(secret)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return nil, err
		}

		return h.secrets.Get(execution.Namespace, secret.Name, metaV1.GetOptions{})
	}
	return s, nil
}

func (h *handler) createJob(or []metaV1.OwnerReference, input *Input, runName, runHash, action, sa, namespace string) (*batchV1.Job, error) {
	createEnvForJob(input, action, runName, namespace)

	meta := metaV1.ObjectMeta{
		Name:            "job-" + runName,
		Namespace:       namespace,
		Labels:          map[string]string{"runHash": runHash},
		OwnerReferences: or,
	}

	backOffLimit := int32(3)

	j := &batchV1.Job{
		ObjectMeta: meta,
		Spec: batchV1.JobSpec{
			BackoffLimit: &backOffLimit,
			Template: coreV1.PodTemplateSpec{
				Spec: coreV1.PodSpec{
					ServiceAccountName: sa,
					Containers: []coreV1.Container{
						{
							Name:  "agent",
							Image: input.Image,
							Env:   input.EnvVars,
						},
					},
					RestartPolicy: "OnFailure",
				},
			},
		},
	}

	job, err := h.jobs.Create(j)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return nil, err
		}
		return h.jobs.Get(namespace, j.Name, metaV1.GetOptions{})
	}

	return job, nil
}

func (h *handler) createServiceAccount(name, namespace string) (*coreV1.ServiceAccount, error) {
	meta := metaV1.ObjectMeta{
		Name:      "sa-" + name,
		Namespace: namespace,
	}
	serviceAccount := coreV1.ServiceAccount{
		ObjectMeta: meta,
	}
	sa, err := h.serviceAccounts.Create(&serviceAccount)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return nil, err
		}
		return h.serviceAccounts.Get(namespace, serviceAccount.Name, metaV1.GetOptions{})
	}
	return sa, nil
}

func (h *handler) createClusterRoleBinding(or []metaV1.OwnerReference, name, role, sa, namespace string) (*rbacV1.ClusterRoleBinding, error) {
	meta := metaV1.ObjectMeta{
		Name:            "crb-" + name,
		OwnerReferences: or,
	}
	clusterRoleBinding := rbacV1.ClusterRoleBinding{
		ObjectMeta: meta,
		Subjects: []rbacV1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: namespace,
			},
		},
		RoleRef: rbacV1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     role,
		},
	}

	rb, err := h.clusterRoleBindings.Create(&clusterRoleBinding)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return nil, err
		}
		return h.clusterRoleBindings.Get(clusterRoleBinding.Name, metaV1.GetOptions{})
	}
	return rb, nil
}

// updateOwnerReference ties the passed in objs to a job
func (h *handler) updateOwnerReference(job *batchV1.Job, objs []interface{}, namespace string) error {
	or := []metaV1.OwnerReference{
		{
			APIVersion: "batch/v1",
			Kind:       "Job",
			Name:       job.Name,
			UID:        job.UID,
		},
	}

	var err error

	for _, obj := range objs {
		switch v := obj.(type) {
		case *v1.Execution:
			err = tryUpdate(func() error {
				execution, err := h.executions.Get(namespace, v.Name, metaV1.GetOptions{})
				if err != nil {
					return err
				}
				execution.OwnerReferences = or
				_, err = h.executions.Update(execution)
				if err != nil {
					return err
				}
				return nil
			})
		case *coreV1.Secret:
			err = tryUpdate(func() error {
				secret, err := h.secrets.Get(namespace, v.Name, metaV1.GetOptions{})
				if err != nil {
					return err
				}
				secret.OwnerReferences = or

				_, err = h.secrets.Update(secret)
				if err != nil {
					return err
				}
				return nil
			})
		case *coreV1.ServiceAccount:
			err = tryUpdate(func() error {
				sa, err := h.serviceAccounts.Get(namespace, v.Name, metaV1.GetOptions{})
				if err != nil {
					return err
				}
				sa.OwnerReferences = or

				_, err = h.serviceAccounts.Update(sa)
				if err != nil {
					return err
				}
				return nil
			})
		case *rbacV1.ClusterRole:
			err = tryUpdate(func() error {
				role, err := h.clusterRoles.Get(v.Name, metaV1.GetOptions{})
				if err != nil {
					return err
				}
				role.OwnerReferences = or

				_, err = h.clusterRoles.Update(role)
				if err != nil {
					return err
				}
				return nil
			})
		case *rbacV1.ClusterRoleBinding:
			err = tryUpdate(func() error {
				binding, err := h.clusterRoleBindings.Get(v.Name, metaV1.GetOptions{})
				if err != nil {
					return err
				}
				binding.OwnerReferences = or

				_, err = h.clusterRoleBindings.Update(binding)
				if err != nil {
					return err
				}
				return nil
			})

		default:
			return errors.New("unknown type")
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func combineVars(input *Input) map[string]string {
	vars := map[string]string{}

	for _, config := range input.Configs {
		for k, v := range config.Data {
			vars[k] = v
		}
	}

	for _, secret := range input.Secrets {
		for k, v := range secret.Data {
			vars[k] = string(v)
		}
	}

	return vars
}

func createEnvForJob(input *Input, action, runName, namespace string) {
	envVars := []coreV1.EnvVar{
		{
			Name:  "TF_IN_AUTOMATION",
			Value: "true",
		},
		{
			Name:  "EXECUTOR_ACTION",
			Value: action,
		},
		{
			Name:  "EXECUTOR_RUN_NAME",
			Value: runName,
		},
		{
			Name:  "EXECUTOR_NAMESPACE",
			Value: namespace,
		},
	}

	input.EnvVars = append(input.EnvVars, envVars...)
}

func getCombinedVars(state *v1.State, input *Input) map[string]string {
	combinedVars := combineVars(input)
	combinedVars["key"] = state.Name

	return combinedVars
}
func createRunHash(state *v1.State, input *Input, action string) string {
	return generateRunHash(state, getCombinedVars(state, input), input.Module.Status.ContentHash, action)
}

func generateRunHash(state *v1.State, vars map[string]string, h string, a string) string {
	varHash := digest.SHA256Map(vars)

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, state.Spec.Version)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}

	hash := sha256.New()
	if _, err := hash.Write([]byte(varHash)); err != nil {
		logrus.Error("Failed to write to digest")
	}
	if _, err := hash.Write([]byte(h)); err != nil {
		logrus.Error("Failed to write to digest")
	}
	if _, err := hash.Write(buf.Bytes()); err != nil {
		logrus.Error("Failed to write to digest")
	}
	if _, err := hash.Write([]byte(a)); err != nil {
		logrus.Error("Failed to write to digest")
	}

	encoding := hex.EncodeToString(hash.Sum(nil))[:10]

	return encoding
}

// tryUpdate runs the input func and if the error returned is a conflict error
// from k8s it will sleep and attempt to run the func again. This is useful
// when attempting to update an object.
func tryUpdate(f func() error) error {
	timeout := 100
	for i := 0; i <= 3; i++ {
		err := f()
		if err != nil {
			if k8sError.IsConflict(err) {
				time.Sleep(time.Duration(timeout) * time.Millisecond)
				timeout *= 2
				continue
			}
			return err
		}
	}
	return nil
}
