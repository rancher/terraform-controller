package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"

	"time"

	"github.com/pkg/errors"
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
func (h *handler) deployCreate(execution *v1.Execution, input *Input, action string) (string, error) {
	logrus.Info("deploy create")
	combinedVars := combineVars(input)
	// Always set the key for the k8s backend
	combinedVars["key"] = execution.Name

	jsonVars, err := json.Marshal(combinedVars)

	logrus.Info(string(jsonVars))

	if err != nil {
		return "", err
	}

	name := createExecRunAndSecretName(execution, combinedVars, input.Module.Status.ContentHash)

	match, err := h.runsMatch(execution, input, jsonVars)
	if err != nil {
		return "", err
	}

	if match {
		return execution.Status.ExecutionRunName, nil
	}

	namespace := execution.Namespace

	or := []metaV1.OwnerReference{
		metaV1.OwnerReference{
			APIVersion: "terraform-controller.cattle.io/v1",
			Kind:       "Execution",
			Name:       execution.Name,
			UID:        execution.UID,
		},
	}

	logrus.Debug("Create - Creating executionRun")
	_, err = h.createExecutionRun(or, execution, name, input)
	if err != nil {
		return "", err
	}

	logrus.Debug("Create - Creating secret")
	_, err = h.createSecretForVariablesFile(or, name, execution, jsonVars)
	if err != nil {
		return "", err
	}

	logrus.Debug("Create - Creating serviceAccount")
	sa, err := h.createServiceAccount(or, name, namespace)
	if err != nil {
		return "", err
	}

	logrus.Debug("Create - Creating clusterRoleBinding")
	rb, err := h.createClusterRoleBinding(or, name, "cluster-admin", sa.Name, namespace)
	if err != nil {
		return "", err
	}

	logrus.Debug("Create - Creating job")
	job, err := h.createJob(or, input, name, action, sa.Name, namespace)
	if err != nil {
		return "", err
	}

	logrus.Debug("CreateUpdating owner references")
	err = h.updateOwnerReference(job, []interface{}{sa, rb}, namespace)
	if err != nil {
		return "", err
	}

	logrus.Infof("Deployed create job for execution %v", execution.Name)

	return name, nil
}

// deployCreate creates all resources for the job to run terraform destroy
func (h *handler) deployDestroy(execution *v1.Execution, input *Input, action string) (string, error) {
	combinedVars := combineVars(input)
	// Always set the key for the k8s backend
	combinedVars["key"] = execution.Name

	jsonVars, err := json.Marshal(combinedVars)
	if err != nil {
		return "", err
	}

	name := createExecRunAndSecretName(execution, combinedVars, input.Module.Status.ContentHash)
	name = name + "-destroy"

	namespace := execution.Namespace

	or := []metaV1.OwnerReference{}

	logrus.Debug("Destroy - Creating executionRun")
	_, err = h.createExecutionRun(or, execution, name, input)
	if err != nil {
		return "", err
	}

	logrus.Debug("Destroy - Creating secret")
	secret, err := h.createSecretForVariablesFile(or, name, execution, jsonVars)
	if err != nil {
		return "", err
	}

	logrus.Debug("Destroy - Creating serviceAccount")
	sa, err := h.createServiceAccount(or, name, namespace)
	if err != nil {
		return "", err
	}

	logrus.Debug("Destroy - Creating clusterRoleBinding")
	rb, err := h.createClusterRoleBinding(or, name, "cluster-admin", sa.Name, namespace)
	if err != nil {
		return "", err
	}

	logrus.Debug("Destroy - Creating job")
	job, err := h.createJob(or, input, name, action, sa.Name, namespace)
	if err != nil {
		return "", err
	}

	logrus.Debug("Destroy - Updating owner references")
	err = h.updateOwnerReference(job, []interface{}{sa, rb, secret}, namespace)
	if err != nil {
		return "", err
	}

	logrus.Infof("Deployed destroy job for execution %v", execution.Name)

	return name, nil
}

// runsMatch checks the previous run with the incoming to determine if the job should be executed again
func (h *handler) runsMatch(execution *v1.Execution, input *Input, jsonVars []byte) (bool, error) {
	if execution.Status.ExecutionRunName == "" {
		return false, nil
	}

	prevExecutionRun, err := h.executionRuns.Get(execution.ObjectMeta.Namespace, execution.Status.ExecutionRunName, metaV1.GetOptions{})
	if err != nil {
		if !k8sError.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}

	if prevExecutionRun.Spec.ContentHash != input.Module.Status.ContentHash {
		return false, nil
	}

	if execution.Spec.Version != prevExecutionRun.Spec.ExecutionVersion {
		return false, nil
	}

	varChange, err := h.varsChanged(jsonVars, prevExecutionRun)
	if err != nil {
		return false, err
	}

	if varChange {
		return false, nil
	}

	// Looks like everything is the same
	return true, nil
}

// varsChanged returns true if the vars have changed since the last run
func (h *handler) varsChanged(vars []byte, run *v1.ExecutionRun) (bool, error) {
	oldSecret, err := h.secrets.Get(run.ObjectMeta.Namespace, run.Spec.SecretName, metaV1.GetOptions{})
	if err != nil {
		if !k8sError.IsNotFound(err) {
			return false, err
		}
		return true, nil
	}

	// Compare the current variables that would be passed to the job to the previous run
	if string(vars) == string(oldSecret.Data["varFile"]) {
		return false, nil
	}

	return true, nil
}

func (h *handler) createExecutionRun(
	or []metaV1.OwnerReference,
	execution *v1.Execution,
	name string,
	input *Input,
) (*v1.ExecutionRun, error) {
	execRun := &v1.ExecutionRun{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       execution.Namespace,
			OwnerReferences: or,
			Annotations:     map[string]string{"approved": ""},
		},
		Spec: v1.ExecutionRunSpec{
			ExecutionName:    execution.Name,
			AutoConfirm:      execution.Spec.AutoConfirm,
			SecretName:       "s-" + name,
			Content:          input.Module.Status.Content,
			ContentHash:      input.Module.Status.ContentHash,
			ExecutionVersion: execution.Spec.Version,
		},
	}

	run, err := h.executionRuns.Create(execRun)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return nil, err
		}

		return h.executionRuns.Get(execution.Namespace, name, metaV1.GetOptions{})

	}
	return run, nil
}

func (h *handler) createSecretForVariablesFile(or []metaV1.OwnerReference, name string, execution *v1.Execution, vars []byte) (*coreV1.Secret, error) {
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

func (h *handler) createJob(or []metaV1.OwnerReference, input *Input, runName, action, sa, namespace string) (*batchV1.Job, error) {
	createEnvForJob(input, action, runName, namespace)

	meta := metaV1.ObjectMeta{
		Name:      "job-" + runName,
		Namespace: namespace,

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
						coreV1.Container{
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

func (h *handler) createServiceAccount(or []metaV1.OwnerReference, name, namespace string) (*coreV1.ServiceAccount, error) {
	meta := metaV1.ObjectMeta{
		Name:            "sa-" + name,
		Namespace:       namespace,
		OwnerReferences: or,
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
			rbacV1.Subject{
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
		metaV1.OwnerReference{
			APIVersion: "batch/v1",
			Kind:       "Job",
			Name:       job.Name,
			UID:        job.UID,
		},
	}

	var err error

	for _, obj := range objs {
		switch v := obj.(type) {
		case *v1.ExecutionRun:
			err = tryUpdate(func() error {
				run, err := h.executionRuns.Get(namespace, v.Name, metaV1.GetOptions{})
				if err != nil {
					return errors.WithMessage(err, "failed executionRun get")
				}
				run.OwnerReferences = or
				_, err = h.executionRuns.Update(run)
				if err != nil {
					return errors.WithMessage(err, "failed executionRun update")
				}
				return nil
			})
		case *coreV1.Secret:
			err = tryUpdate(func() error {
				secret, err := h.secrets.Get(namespace, v.Name, metaV1.GetOptions{})
				if err != nil {
					return errors.WithMessage(err, "failed secret get")
				}
				secret.OwnerReferences = or

				_, err = h.secrets.Update(secret)
				if err != nil {
					return errors.WithMessage(err, "failed secret update")
				}
				return nil
			})
		case *coreV1.ServiceAccount:
			err = tryUpdate(func() error {
				sa, err := h.serviceAccounts.Get(namespace, v.Name, metaV1.GetOptions{})
				if err != nil {
					return errors.WithMessage(err, "failed serviceAccount get")
				}
				sa.OwnerReferences = or

				_, err = h.serviceAccounts.Update(sa)
				if err != nil {
					return errors.WithMessage(err, "failed serviceAccount update")
				}
				return nil
			})
		case *rbacV1.ClusterRole:
			err = tryUpdate(func() error {
				role, err := h.clusterRoles.Get(v.Name, metaV1.GetOptions{})
				if err != nil {
					return errors.WithMessage(err, "failed clusterRole get")
				}
				role.OwnerReferences = or

				_, err = h.clusterRoles.Update(role)
				if err != nil {
					return errors.WithMessage(err, "failed clusterRole update")
				}
				return nil
			})
		case *rbacV1.ClusterRoleBinding:
			err = tryUpdate(func() error {
				binding, err := h.clusterRoleBindings.Get(v.Name, metaV1.GetOptions{})
				if err != nil {
					return errors.WithMessage(err, "failed clusterRoleBinding get")
				}
				binding.OwnerReferences = or

				_, err = h.clusterRoleBindings.Update(binding)
				if err != nil {
					return errors.WithMessage(err, "failed clusterRoleBinding update")
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
		coreV1.EnvVar{
			Name:  "TF_IN_AUTOMATION",
			Value: "true",
		},
		coreV1.EnvVar{
			Name:  "EXECUTOR_ACTION",
			Value: action,
		},
		coreV1.EnvVar{
			Name:  "EXECUTOR_RUN_NAME",
			Value: runName,
		},
		coreV1.EnvVar{
			Name:  "EXECUTOR_NAMESPACE",
			Value: namespace,
		},
	}

	input.EnvVars = append(input.EnvVars, envVars...)
}

func createExecRunAndSecretName(execution *v1.Execution, vars map[string]string, h string) string {
	varHash := digest.SHA256Map(vars)

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, execution.Spec.Version)
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

	encoding := hex.EncodeToString(hash.Sum(nil))[:10]

	return execution.ObjectMeta.Name + "-" + encoding
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
