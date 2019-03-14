package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/norman/controller"
	batchv1 "github.com/rancher/terraform-operator/types/apis/batch/v1"
	"github.com/rancher/terraform-operator/types/apis/client"
	coreclient "github.com/rancher/terraform-operator/types/apis/core/v1"
	rbacv1 "github.com/rancher/terraform-operator/types/apis/rbac.authorization.k8s.io/v1"
	"github.com/rancher/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
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
	DefaultExecutorImage = "rancher/terraform-operator-executor"
)

func Register(ctx context.Context, ns string, client *client.MasterClient) error {
	e := &executionLifecycle{
		clusterRoles:        client.RBAC.ClusterRoles(""),
		clusterRoleBindings: client.RBAC.ClusterRoleBindings(""),
		configMapLister:     client.Core.ConfigMaps("").Controller().Lister(),
		executions:          client.Operator.Executions(""),
		executionLister:     client.Operator.Executions("").Controller().Lister(),
		executionRuns:       client.Operator.ExecutionRuns(""),
		executionRunLister:  client.Operator.ExecutionRuns("").Controller().Lister(),
		jobs:                client.Batch.Jobs(""),
		moduleLister:        client.Operator.Modules("").Controller().Lister(),
		secrets:             client.Core.Secrets(""),
		secretsLister:       client.Core.Secrets("").Controller().Lister(),
		serviceAccounts:     client.Core.ServiceAccounts(""),
	}

	client.Operator.Executions("").AddLifecycle(ctx, "execution-controller", e)
	return nil
}

type executionLifecycle struct {
	clusterRoles        rbacv1.ClusterRoleInterface
	clusterRoleBindings rbacv1.ClusterRoleBindingInterface
	configMapLister     coreclient.ConfigMapLister
	executions          v1.ExecutionInterface
	executionLister     v1.ExecutionLister
	executionRuns       v1.ExecutionRunInterface
	executionRunLister  v1.ExecutionRunLister
	jobs                batchv1.JobInterface
	moduleLister        v1.ModuleLister
	secrets             coreclient.SecretInterface
	secretsLister       coreclient.SecretLister
	serviceAccounts     coreclient.ServiceAccountInterface
}

func (e *executionLifecycle) Create(obj *v1.Execution) (runtime.Object, error) {
	if obj.Spec.Version < 1 {
		obj.Spec.Version = 1
	}
	if obj.Spec.Image == "" {
		// TODO: Need a real default image
		obj.Spec.Image = fmt.Sprintf("%s:latest", DefaultExecutorImage)
	}
	return e.executions.Update(obj)
}

func (e *executionLifecycle) Remove(obj *v1.Execution) (runtime.Object, error) {
	input, ok, err := e.gatherInput(obj)
	if !ok {
		v1.ExecutionConditionMissingInfo.True(obj)
		return e.executions.Update(obj)
	}
	if err != nil {
		return obj, err
	}

	v1.ExecutionConditionMissingInfo.False(obj)

	if obj.Spec.DestroyOnDelete {
		var runName string
		_, err = v1.ExecutionConditionDestroyJobDeployed.DoUntilTrue(obj, func() (runtime.Object, error) {
			runName, err = e.deployDestroy(obj, input, ActionDestroy)
			if err != nil {
				return obj, err
			}

			if obj.Status.ExecutionRunName != runName {
				obj.Status.ExecutionRunName = runName
				obj, err = e.executions.Update(obj)
				if err != nil {
					return obj, errors.WithMessage(err, "updating executionRun on execution")
				}
			}
			return obj, nil
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

		run, err := e.executionRunLister.Get(obj.Namespace, runName)
		if err != nil {
			return obj, errors.WithMessage(err, "error getting execution run")
		}

		if v1.ExecutionRunConditionApplied.IsTrue(run) {
			err = e.executionRuns.DeleteNamespaced(run.Namespace, run.Name, &metaV1.DeleteOptions{})
			if err != nil {
				if !k8serrors.IsNotFound(err) {
					return obj, err
				}
			}
			return obj, nil
		}

		if !v1.ExecutionConditionWatchRunning.IsTrue(obj) {
			go e.watchDestroyRun(obj, run)
			v1.ExecutionConditionWatchRunning.True(obj)
		}

		return obj, &controller.ForgetError{}

	}
	return obj, nil
}

func (e *executionLifecycle) Updated(obj *v1.Execution) (runtime.Object, error) {
	input, ok, err := e.gatherInput(obj)
	if !ok {
		v1.ExecutionConditionMissingInfo.True(obj)
		return e.executions.Update(obj)
	}
	if err != nil {
		return obj, err
	}

	v1.ExecutionConditionMissingInfo.False(obj)

	return v1.ExecutionConditionJobDeployed.Track(obj, e.executions, func() (runtime.Object, error) {
		runName, err := e.deployCreate(obj, input, ActionCreate)
		if err != nil {
			return obj, err
		}

		if obj.Status.ExecutionRunName != runName {
			obj.Status.ExecutionRunName = runName
			return e.executions.Update(obj)
		}

		return obj, nil
	})
}

// watchDestroyRun checks the executionRun for the Applied condition, once set
// terraform destroy was run so the execution can be requeued for deletion.
func (e *executionLifecycle) watchDestroyRun(execution *v1.Execution, run *v1.ExecutionRun) {
	for {
		r, err := e.executionRunLister.Get(run.Namespace, run.Name)
		if err != nil {
			return
		}
		if v1.ExecutionRunConditionApplied.IsTrue(r) {
			e.executions.Controller().Enqueue(execution.Namespace, execution.Name)
			return
		}
		time.Sleep(2 * time.Second)
	}
}
