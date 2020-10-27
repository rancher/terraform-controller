package state

import (
	"context"
	"fmt"
	"github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	batchv1 "github.com/rancher/wrangler/pkg/generated/controllers/batch/v1"
	corev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	rbacv1 "github.com/rancher/wrangler/pkg/generated/controllers/rbac/v1"
	"github.com/sirupsen/logrus"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	//ActionCreate for terraform
	ActionCreate = "create"
	//ActionDestroy for terraform
	ActionDestroy = "destroy"
	//Default Image
	DefaultExecutorImage = "rancher/terraform-controller-executor"
)

func NewHandler(
	ctx context.Context,
	modules tfv1.ModuleController,
	states tfv1.StateController,
	executions tfv1.ExecutionController,
	clusterRoles rbacv1.ClusterRoleController,
	clusterRoleBindings rbacv1.ClusterRoleBindingController,
	secrets corev1.SecretController,
	configMaps corev1.ConfigMapController,
	serviceAccounts corev1.ServiceAccountController,
	jobs batchv1.JobController,
) *handler {
	return &handler{
		ctx:                 ctx,
		modules:             modules,
		states:              states,
		executions:          executions,
		clusterRoles:        clusterRoles,
		clusterRoleBindings: clusterRoleBindings,
		secrets:             secrets,
		configMaps:          configMaps,
		serviceAccounts:     serviceAccounts,
		jobs:                jobs,
	}
}

type handler struct {
	ctx                 context.Context
	modules             tfv1.ModuleController
	states              tfv1.StateController
	executions          tfv1.ExecutionController
	clusterRoles        rbacv1.ClusterRoleController
	clusterRoleBindings rbacv1.ClusterRoleBindingController
	secrets             corev1.SecretController
	configMaps          corev1.ConfigMapController
	serviceAccounts     corev1.ServiceAccountController
	jobs                batchv1.JobController
}

func (h *handler) OnChange(key string, obj *v1.State) (*v1.State, error) {
	logrus.Debugf("State On Change Handler %s", key)
	if obj == nil {
		return nil, nil
	}

	if obj.DeletionTimestamp != nil {
		logrus.Debugf("object %s is marked for deletion", key)
		return nil, nil
	}

	input, ok, err := h.gatherInput(obj)
	if err != nil {
		return obj, err
	}

	if !ok {
		v1.ExecutionConditionMissingInfo.SetStatusBool(obj, ok)
		logrus.Debug("missing info")
		return h.states.Update(obj)
	}

	if v1.StateConditionJobDeployed.IsTrue(obj) && obj.Status.LastRunHash != "" {
		logrus.Debugf("job already running %s, checking execution", obj.Status.LastRunHash)
		execution, err := h.executions.Get(obj.Namespace, obj.Status.ExecutionName, metaV1.GetOptions{})
		if err != nil {
			logrus.Errorf("error while retrieving execution %v", err)
			return obj, err
		}
		if v1.ExecutionRunConditionApplied.IsTrue(execution) {
			logrus.Debugf("execution is complete. setting required conditions on state")
			v1.StateConditionJobDeployed.False(obj)
			obj.Status.ExecutionName = ""
			obj, err = h.states.Update(obj)
			if err != nil {
				logrus.Error(err)
				return obj, err
			}
			return obj, nil // return nil which will remove this state because the execution is done
		}
	}

	if obj.Spec.Version < 1 {
		obj.Spec.Version = 1
	}

	if obj.Spec.Version < 1 {
		obj.Spec.Version = 1
	}

	runHash := createRunHash(obj, input, ActionCreate)
	if runHash == obj.Status.LastRunHash {
		logrus.Debugf("last run hash is %s", runHash)
		return obj, nil
	}

	//running an execution
	obj, err = h.states.Update(obj)
	if err != nil {
		logrus.Error(err)
		return obj, nil
	}

	logrus.Debug("lock acquired with new hash")

	if obj.Spec.Image == "" {
		obj.Spec.Image = fmt.Sprintf("%s:latest", DefaultExecutorImage)
	}

	//new execution if none running
	exec, err := h.deployCreate(obj, input)
	if err != nil {
		logrus.Debugf("failed to create execution for %s: %s", obj.Name, err)
		return obj, err
	}

	v1.StateConditionJobDeployed.True(obj)
	obj.Status.ExecutionName = exec.Name
	obj.Status.LastRunHash = runHash

	return h.states.Update(obj)
}

func (h *handler) OnRemove(key string, obj *v1.State) (*v1.State, error) {
	logrus.Debugf("State On Remove Handler %s", key)
	input, ok, err := h.gatherInput(obj)
	if err != nil {
		logrus.Debug("error gathering input")
		return obj, err
	}
	if !ok {
		v1.ExecutionConditionMissingInfo.SetStatusBool(obj, ok)
		state, err := h.states.Update(obj)
		if err != nil {
			return state, err
		}

		return state, fmt.Errorf("missing info and can not run destroy")
	}

	if !obj.Spec.DestroyOnDelete || v1.StateConditionDestroyed.IsTrue(obj) {
		return obj, nil
	}

	v1.ExecutionConditionMissingInfo.False(obj)

	if v1.StateConditionJobDeployed.IsTrue(obj) && obj.Status.LastRunHash != "" {
		logrus.Debugf("remove job already running %s, checking execution", obj.Status.LastRunHash)
		execution, err := h.executions.Get(obj.Namespace, obj.Status.ExecutionName, metaV1.GetOptions{})
		if err != nil {
			logrus.Errorf("error while retrieving execution %v", err)
		}
		if v1.ExecutionRunConditionApplied.IsTrue(execution) {
			logrus.Debugf("execution is complete. cleaning up")
			return obj, nil // return nil which will remove this state because the execution is done
		}
		return obj, fmt.Errorf("execution job for remove has been deployed and is not done yet")
	}

	runHash := createRunHash(obj, input, ActionDestroy)
	if runHash == obj.Status.LastRunHash && v1.StateConditionJobDeployed.IsTrue(obj) {
		logrus.Debug("hashes the same and job already deployed, nothing to do")
		return obj, fmt.Errorf("benign error, hashes are same")
	}

	//running an execution
	logrus.Debug("acquire lock")
	obj.Status.LastRunHash = runHash
	obj, err = h.states.Update(obj)
	if err != nil {
		logrus.Debug("lock failed")
		logrus.Debug(err)
		return obj, err
	}

	logrus.Debug("deploying destroy job")

	//no job running, try and run a destroy
	exec, err := h.deployDestroy(obj, input)
	if err != nil {
		logrus.Error(err)
		return obj, err
	}

	obj.Status.ExecutionName = exec.Name
	v1.StateConditionJobDeployed.True(obj)

	_, err = h.states.Update(obj)
	if err != nil {
		return obj, err
	}

	//return error because no error will clear finalizer even if job did not complete.
	return obj, fmt.Errorf("execution for destroy has been run")
}
