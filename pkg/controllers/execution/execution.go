package execution

import (
	"context"

	"github.com/rancher/kerraform/pkg/controllers/execution/deploy"

	coreclient "github.com/rancher/kerraform/types/apis/core/v1"
	"github.com/rancher/kerraform/types/apis/kerraform.cattle.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Register(ctx context.Context, ns string, coreclient coreclient.Interface, client v1.Interface) error {
	e := &executionLifecycle{}

	client.Executions(ns).AddLifecycle(ctx, "execution-controller", e)
}

type executionLifecycle struct {
	executions      v1.ExecutionInterface
	executionLister v1.ExecutionLister
	secretsLister   coreclient.SecretLister
	moduleLister    v1.ModuleLister
	configMapLister coreclient.ConfigMapLister
}

func (e *executionLifecycle) Create(obj *v1.Execution) (runtime.Object, error) {
	return obj, nil
}

func (e *executionLifecycle) Remove(obj *v1.Execution) (runtime.Object, error) {
	return obj, deploy.Remove(obj)
}

func (e *executionLifecycle) Updated(obj *v1.Execution) (runtime.Object, error) {
	input, ok, err := e.gatherInput(obj)
	if !ok || err != nil {
		return obj, err
	}

	return v1.ExecutionConditionJobDeployed.Track(obj, e.executions, func() (runtime.Object, error) {
		runName, err := deploy.Deploy(obj, input)
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
