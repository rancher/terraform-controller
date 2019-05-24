package state

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	batchv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/batch/v1"
	corev1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	rbacv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func (h *handler) OnRemove(key string, obj *v1.State) (*v1.State, error) {
	input, ok, err := h.gatherInput(obj)
	if !ok {
		v1.ExecutionConditionMissingInfo.True(obj)
		return h.states.Update(obj)
	}
	if err != nil {
		return obj, err
	}

	v1.ExecutionConditionMissingInfo.False(obj)

	if !obj.Spec.DestroyOnDelete {
		return obj, nil
	}

	var runName string
	err = v1.ExecutionConditionDestroyJobDeployed.DoUntilTrue(obj, func() (runtime.Object, error) {
		runName, err = h.deployDestroy(obj, input, ActionDestroy)
		if err != nil {
			return obj, err
		}

		if obj.Status.ExecutionName != runName {
			obj.Status.ExecutionName = runName
			obj, err = h.states.Update(obj)
			if err != nil {
				return obj, errors.WithMessage(err, "updating executionRun on execution")
			}
		}
		return h.states.Update(obj)
	})
	if err != nil {
		return obj, errors.WithMessage(err, "track error")
	}

	if runName == "" {
		combinedVars := combineVars(input)
		combinedVars["key"] = obj.Name
		name := createExecRunAndSecretName(obj, combinedVars, input.Module.Status.ContentHash)
		runName = name + "-destroy"
	}

	run, err := h.states.Get(obj.Namespace, runName, metaV1.GetOptions{})
	if err != nil {
		return obj, errors.WithMessage(err, "error getting execution run")
	}

	if v1.ExecutionRunConditionApplied.IsTrue(run) {
		err = h.states.Delete(run.Namespace, run.Name, &metaV1.DeleteOptions{})
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return obj, err
			}
		}
		return obj, nil
	}

	if !v1.ExecutionConditionWatchRunning.IsTrue(obj) {
		go h.watchDestroyRun(obj, run)
		v1.ExecutionConditionWatchRunning.True(obj)
	}

	return h.states.Update(obj)

}

func (h *handler) OnChange(key string, obj *v1.State) (*v1.State, error) {
	if obj == nil {
		return nil, nil
	}
	if obj.Spec.Version < 1 {
		obj.Spec.Version = 1
	}
	if obj.Spec.Image == "" {
		obj.Spec.Image = fmt.Sprintf("%s:dev", DefaultExecutorImage)
	}

	input, ok, err := h.gatherInput(obj)

	if !ok {
		v1.ExecutionConditionMissingInfo.True(obj)
		return h.states.Update(obj)
	}
	if err != nil {
		return obj, err
	}

	v1.ExecutionConditionMissingInfo.False(obj)
	v1.ExecutionConditionJobDeployed.False(obj)

	return tfv1.UpdateStateOnChange(func(state runtime.Object) (runtime.Object, error) {
		v1.ExecutionConditionJobDeployed.True(obj)
		return h.states.Update(state.(*v1.State))
	}, func(key string, obj *v1.State) (*v1.State, error) {
		runName, err := h.deployCreate(obj, input, ActionCreate)
		logrus.Infof("runName: %s", runName)
		if err != nil {
			return obj, err
		}

		if obj.Status.ExecutionName != runName {
			obj.Status.ExecutionName = runName
			return h.states.Update(obj)
		}
		return obj, nil
	})(key, obj)
}

// watchDestroyRun checks the executionRun for the Applied condition, once set
// terraform destroy was run so the execution can be requeued for deletion.
func (h *handler) watchDestroyRun(state *v1.State, run *v1.State) {
	for {
		r, err := h.executions.Get(run.Namespace, run.Name, metaV1.GetOptions{})
		if err != nil {
			return
		}
		if v1.ExecutionRunConditionApplied.IsTrue(r) {
			h.states.Enqueue(state.Namespace, state.Name)
			return
		}
		time.Sleep(2 * time.Second)
	}
}
