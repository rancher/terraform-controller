package execution

import (
	"context"
	"fmt"

	batchv1 "github.com/ibuildthecloud/terraform-operator/types/apis/batch/v1"
	"github.com/ibuildthecloud/terraform-operator/types/apis/client"
	coreclient "github.com/ibuildthecloud/terraform-operator/types/apis/core/v1"
	rbacv1 "github.com/ibuildthecloud/terraform-operator/types/apis/rbac.authorization.k8s.io/v1"
	"github.com/ibuildthecloud/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	//ActionCreate for terraform
	ActionCreate = "create"
	//ActionDestroy for terraform
	ActionDestroy = "destroy"
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

	client.Operator.Executions(ns).AddLifecycle(ctx, "execution-controller", e)
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
	fmt.Println("Create")
	if obj.Spec.Version < 1 {
		obj.Spec.Version = 1
	}
	return e.executions.Update(obj)
}

func (e *executionLifecycle) Remove(obj *v1.Execution) (runtime.Object, error) {
	input, ok, err := e.gatherInput(obj)
	if !ok || err != nil {
		fmt.Printf("gettin here: %v, %v", ok, err)
		return obj, err
	}

	return obj, e.deployDestroy(obj, input, ActionDestroy)
}

func (e *executionLifecycle) Updated(obj *v1.Execution) (runtime.Object, error) {
	input, ok, err := e.gatherInput(obj)
	if !ok || err != nil {
		fmt.Printf("gettin here: %v, %v", ok, err)
		return obj, err
	}

	return v1.ExecutionConditionJobDeployed.Track(obj, e.executions, func() (runtime.Object, error) {
		runName, err := e.deployCreate(obj, input, ActionCreate)
		if err != nil {
			return obj, err
		}

		if obj.Status.ExecutionRunName != runName {
			obj = obj.DeepCopy()
			obj.Status.ExecutionRunName = runName
			return e.executions.Update(obj)
		}

		return obj, nil
	})
}
