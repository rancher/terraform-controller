package state

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	batchv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/batch/v1"
	corev1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	rbacv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/rbac/v1"
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
	logrus.Debug("State On Change Handler")
	if obj == nil {
		return nil, nil
	}

	if obj.DeletionTimestamp != nil {
		return nil, nil
	}

	input, ok, err := h.gatherInput(obj)
	if err != nil {
		return obj, err
	}

	if !ok {
		v1.ExecutionConditionMissingInfo.SetStatus(obj, err.Error())
		return h.states.Update(obj)
	}

	if v1.StateConditionJobDeployed.IsTrue(obj) && obj.Status.LastRunHash != "" {
		return obj, nil
	}

	runHash := createRunHash(obj, input, ActionCreate)
	if runHash == obj.Status.LastRunHash {
		return obj, nil
	}

	//running an execution
	obj.Status.LastRunHash = runHash
	obj, err = h.states.Update(obj)
	if err != nil {
		logrus.Debug(err)
		return obj, nil
	}

	logrus.Debug("lock acquired with new hash")

	if obj.Spec.Version < 1 {
		obj.Spec.Version = 1
	}
	if obj.Spec.Image == "" {
		obj.Spec.Image = fmt.Sprintf("%s:latest", DefaultExecutorImage)
	}

	//new execution if none running
	exec, err := h.deployCreate(obj, input)
	if err != nil {
		logrus.Debugf("failed to create execution: %s", err)
		return obj, err
	}

	v1.StateConditionJobDeployed.True(obj)
	obj.Status.ExecutionName = exec.Name

	//execution was run, find it and set a watcher to clean up when it's done.
	execution, err := h.executions.Get(obj.Namespace, obj.Status.ExecutionName, metaV1.GetOptions{})
	if err != nil {
		return obj, errors.WithMessage(err, "error getting execution")
	}

	go h.watchExecution(obj, execution, ActionCreate)

	return h.states.Update(obj)
}

func (h *handler) OnRemove(key string, obj *v1.State) (*v1.State, error) {
	logrus.Debug("State On Remove Handler")
	input, ok, err := h.gatherInput(obj)
	if err != nil {
		logrus.Debug("error gathering input")
		return obj, err
	}
	if !ok {
		v1.ExecutionConditionMissingInfo.SetStatus(obj, err.Error())
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
		return obj, fmt.Errorf("job already running %s", obj.Status.LastRunHash)
	}

	runHash := createRunHash(obj, input, ActionDestroy)
	if runHash == obj.Status.LastRunHash {
		logrus.Debug("hashes the same, nothing to do")
		return obj, fmt.Errorf("job already running %s", obj.Status.LastRunHash)
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

	logrus.Debug("deploy destroy")

	//no job running, try and run a destroy
	exec, err := h.deployDestroy(obj, input)
	if err != nil {
		logrus.Debug(err)
		return obj, err
	}

	obj.Status.ExecutionName = exec.Name
	v1.StateConditionJobDeployed.True(obj)

	//destroy deployed, setup watcher.
	execution, err := h.executions.Get(obj.Namespace, obj.Status.ExecutionName, metaV1.GetOptions{})
	if err != nil {
		logrus.Debug("error getting the execution")
		return obj, errors.WithMessage(err, "error getting execution")
	}

	go h.watchExecution(obj, execution, ActionDestroy)

	//return error because no error will clear finalizer even if job did not complete.
	_, err = h.states.Update(obj)
	if err != nil {
		return obj, err
	}

	return obj, fmt.Errorf("watching execution to complete")
}

func (h *handler) watchExecution(state *v1.State, execution *v1.Execution, action string) {
	logrus.Debugf("About to watch %s", execution.Name)
	maxIterations := 450 //15min
	i := 0
	for {
		logrus.Debugf("Waiting for execution %s.", execution.Name)
		if maxIterations < i {
			return
		}

		exec, err := h.executions.Get(execution.Namespace, execution.Name, metaV1.GetOptions{})
		if err != nil {
			return
		}
		if v1.ExecutionRunConditionApplied.IsTrue(exec) {
			currentState, err := h.states.Get(state.Namespace, state.Name, metaV1.GetOptions{})
			if err != nil {
				logrus.Debug(err)
				return
			}

			logrus.Debugf("Execution %s complete", execution.Name)
			v1.ExecutionConditionWatchRunning.False(currentState)
			v1.StateConditionJobDeployed.False(currentState)
			currentState.Status.ExecutionName = ""
			if ActionDestroy == action {
				logrus.Debug("Setting Destroyed Condition")
				v1.StateConditionDestroyed.True(currentState)
			}

			_, err = h.states.Update(currentState)
			if err != nil {
				logrus.Errorf("Error updating State after watching the job to completion: %s", err)
			}

			h.states.Enqueue(currentState.Namespace, currentState.Name)
			return
		}

		time.Sleep(2 * time.Second)
		i++
	}
}
