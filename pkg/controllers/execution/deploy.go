package execution

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ibuildthecloud/terraform-operator/pkg/digest"
	"github.com/ibuildthecloud/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"github.com/sirupsen/logrus"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Input struct {
	Module     *v1.Module
	Executions map[string]string
	Configs    []*coreV1.ConfigMap
	Secrets    []*coreV1.Secret
}

// prepareForJob returns the executionRun name
func (e *executionLifecycle) prepareForJob(execution *v1.Execution, input *Input) (string, error) {
	fmt.Println("PREPARE")

	combinedVars := combineVars(input)

	jsonVars, err := json.Marshal(combinedVars)
	if err != nil {
		return "", err
	}

	match, err := e.runsMatch(execution, input, jsonVars)
	if err != nil {
		return "", err
	}

	if match {
		return execution.Status.ExecutionRunName, nil
	}

	name := createExecRunAndSecretName(execution, combinedVars, input.Module.Status.ContentHash)
	namespace := execution.Namespace

	or := []metaV1.OwnerReference{
		metaV1.OwnerReference{
			APIVersion: "terraform-operator.cattle.io/v1",
			Kind:       "Execution",
			Name:       execution.Name,
			UID:        execution.UID,
		},
	}

	logrus.Info("Creating executionRun")
	err = e.createExecutionRun(or, execution, name, input)
	if err != nil {
		return "", err
	}

	logrus.Info("Creating secret")
	err = e.createSecretForVariablesFile(or, name, execution, jsonVars)
	if err != nil {
		return "", err
	}

	logrus.Info("Creating serviceAccount")
	sa, err := e.createServiceAccount(or, name, namespace)
	if err != nil {
		return "", err
	}

	logrus.Info("Creating clusterRoleBinding")
	rb, err := e.createClusterRoleBinding(or, name, "cluster-admin", sa.Name, namespace)
	if err != nil {
		return "", err
	}

	logrus.Info("Creating job")
	job, err := e.createJob(or, name, "create", sa.Name, namespace)
	if err != nil {
		return "", err
	}

	logrus.Info("Updating owner references")
	err = e.updateOwnerReference(job, []interface{}{sa, rb}, namespace)
	if err != nil {
		return "", err
	}

	return name, nil
}

func (e *executionLifecycle) removeExecution(execution *v1.Execution) error {
	// TODO: This needs to run the terraform destroy
	// err := e.createJob(sa, "destroy")
	// if err != nil {
	// 	return err
	// }
	return nil
}

// runsMatch checks the previous run with the incoming to determine if the job should be executed again
func (e *executionLifecycle) runsMatch(execution *v1.Execution, input *Input, jsonVars []byte) (bool, error) {
	prevExecutionRun, err := e.executionRunLister.Get(execution.ObjectMeta.Namespace, execution.Status.ExecutionRunName)
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

	varChange, err := e.varsChanged(jsonVars, prevExecutionRun)
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
func (e *executionLifecycle) varsChanged(vars []byte, run *v1.ExecutionRun) (bool, error) {
	oldSecret, err := e.secretsLister.Get(run.ObjectMeta.Namespace, run.Spec.SecretName)
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

func (e *executionLifecycle) createExecutionRun(
	or []metaV1.OwnerReference,
	execution *v1.Execution,
	name string,
	input *Input,
) error {
	execRun := &v1.ExecutionRun{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       execution.Namespace,
			OwnerReferences: or,
		},
		Spec: v1.ExecutionRunSpec{
			ExecutionName:    execution.Name,
			AutoConfirm:      execution.Spec.AutoConfirm,
			SecretName:       name,
			Content:          input.Module.Spec.ModuleContent,
			ContentHash:      input.Module.Status.ContentHash,
			ExecutionVersion: execution.Spec.Version,
		},
	}

	_, err := e.executionRuns.Create(execRun)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func (e *executionLifecycle) createSecretForVariablesFile(or []metaV1.OwnerReference, name string, execution *v1.Execution, vars []byte) error {
	secretData := map[string][]byte{
		"varFile": vars,
	}

	secret := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       execution.Namespace,
			OwnerReferences: or,
		},
		Data: secretData,
	}

	_, err := e.secrets.Create(secret)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func (e *executionLifecycle) createJob(or []metaV1.OwnerReference, runName, action, sa, namespace string) (*batchV1.Job, error) {
	meta := metaV1.ObjectMeta{
		Name:            "job-" + runName,
		Namespace:       namespace,
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
							Name: "agent",
							// TODO: Need image name
							Image: "nginx",
							Env: []coreV1.EnvVar{
								coreV1.EnvVar{
									Name:  "TF_IN_AUTOMATION",
									Value: "true",
								},
								coreV1.EnvVar{
									Name:  "TF_ACTION",
									Value: action,
								},
								coreV1.EnvVar{
									Name:  "TF_RUN_NAME",
									Value: runName,
								},
							},
							ImagePullPolicy: coreV1.PullAlways,
						},
					},
					RestartPolicy: "OnFailure",
				},
			},
		},
	}

	job, err := e.jobs.Create(j)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return nil, err
		}
		return e.jobs.GetNamespaced(namespace, j.Name, metaV1.GetOptions{})
	}
	return job, nil
}

// TODO: This isn't used yet, 'a'dmin' will be replaced with this customized role for the job
func (e *executionLifecycle) createClusterRole(name string) (*rbacV1.ClusterRole, error) {
	meta := metaV1.ObjectMeta{
		Name: "cr-" + name,
	}

	rules := []rbacV1.PolicyRule{
		// This is needed to check for cattle-system, remove finalizers and delete
		rbacV1.PolicyRule{
			Verbs:     []string{"list", "get", "update", "delete"},
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
		},
		rbacV1.PolicyRule{
			Verbs:     []string{"list", "get", "delete"},
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"roles", "rolebindings", "clusterroles", "clusterrolebindings"},
		},
		// The job is going to delete itself after running to trigger ownerReference
		// cleanup of the clusterRole, serviceAccount and clusterRoleBinding
		rbacV1.PolicyRule{
			Verbs:     []string{"list", "get", "delete"},
			APIGroups: []string{"batch"},
			Resources: []string{"jobs"},
		},
	}
	clusterRole := rbacV1.ClusterRole{
		ObjectMeta: meta,
		Rules:      rules,
	}
	_, err := e.clusterRoles.Create(&clusterRole)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return nil, err
		}
	}
	return nil, nil
}

func (e *executionLifecycle) createServiceAccount(or []metaV1.OwnerReference, name, namespace string) (*coreV1.ServiceAccount, error) {
	meta := metaV1.ObjectMeta{
		Name:            "sa-" + name,
		Namespace:       namespace,
		OwnerReferences: or,
	}
	serviceAccount := coreV1.ServiceAccount{
		ObjectMeta: meta,
	}
	sa, err := e.serviceAccounts.Create(&serviceAccount)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return nil, err
		}
		return e.serviceAccounts.GetNamespaced(namespace, serviceAccount.Name, metaV1.GetOptions{})
	}
	return sa, nil
}

func (e *executionLifecycle) createClusterRoleBinding(or []metaV1.OwnerReference, name, role, sa, namespace string) (*rbacV1.ClusterRoleBinding, error) {
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

	rb, err := e.clusterRoleBindings.Create(&clusterRoleBinding)
	if err != nil {
		if !k8sError.IsAlreadyExists(err) {
			return nil, err
		}
		return e.clusterRoleBindings.Get(clusterRoleBinding.Name, metaV1.GetOptions{})
	}
	return rb, nil
}

func (e *executionLifecycle) updateOwnerReference(job *batchV1.Job, objs []interface{}, namespace string) error {
	or := []metaV1.OwnerReference{
		metaV1.OwnerReference{
			APIVersion: "batch/v1",
			Kind:       "Job",
			Name:       job.Name,
			UID:        job.UID,
		},
	}

	for _, obj := range objs {
		switch v := obj.(type) {
		case *coreV1.ServiceAccount:
			return tryUpdate(func() error {
				role, err := e.serviceAccounts.GetNamespaced(namespace, v.Name, metaV1.GetOptions{})
				if err != nil {
					return err
				}
				role.OwnerReferences = or

				_, err = e.serviceAccounts.Update(role)
				if err != nil {
					return err
				}
				return nil
			})
		case *rbacV1.ClusterRole:
			return tryUpdate(func() error {
				role, err := e.clusterRoles.Get(v.Name, metaV1.GetOptions{})
				if err != nil {
					return err
				}
				role.OwnerReferences = or

				_, err = e.clusterRoles.Update(role)
				if err != nil {
					return err
				}
				return nil
			})
		case *rbacV1.ClusterRoleBinding:
			return tryUpdate(func() error {
				role, err := e.clusterRoleBindings.Get(v.Name, metaV1.GetOptions{})
				if err != nil {
					return err
				}
				role.OwnerReferences = or

				_, err = e.clusterRoleBindings.Update(role)
				if err != nil {
					return err
				}
				return nil
			})

		default:
			return errors.New("unknown type")
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

	for k, v := range input.Executions {
		vars[k] = v
	}

	return vars
}

func createExecRunAndSecretName(execution *v1.Execution, vars map[string]string, h string) string {
	varHash := digest.SHA256Map(vars)

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, execution.Spec.Version)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}

	digest := sha256.New()
	digest.Write([]byte(varHash))
	digest.Write([]byte(h))
	digest.Write(buf.Bytes())

	hash := hex.EncodeToString(digest.Sum(nil))[:10]

	return execution.ObjectMeta.Name + "-" + hash
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
