package v1

import (
	"github.com/rancher/norman/lifecycle"
	"k8s.io/apimachinery/pkg/runtime"
)

type ExecutionRunLifecycle interface {
	Create(obj *ExecutionRun) (runtime.Object, error)
	Remove(obj *ExecutionRun) (runtime.Object, error)
	Updated(obj *ExecutionRun) (runtime.Object, error)
}

type executionRunLifecycleAdapter struct {
	lifecycle ExecutionRunLifecycle
}

func (w *executionRunLifecycleAdapter) HasCreate() bool {
	o, ok := w.lifecycle.(lifecycle.ObjectLifecycleCondition)
	return !ok || o.HasCreate()
}

func (w *executionRunLifecycleAdapter) HasFinalize() bool {
	o, ok := w.lifecycle.(lifecycle.ObjectLifecycleCondition)
	return !ok || o.HasFinalize()
}

func (w *executionRunLifecycleAdapter) Create(obj runtime.Object) (runtime.Object, error) {
	o, err := w.lifecycle.Create(obj.(*ExecutionRun))
	if o == nil {
		return nil, err
	}
	return o, err
}

func (w *executionRunLifecycleAdapter) Finalize(obj runtime.Object) (runtime.Object, error) {
	o, err := w.lifecycle.Remove(obj.(*ExecutionRun))
	if o == nil {
		return nil, err
	}
	return o, err
}

func (w *executionRunLifecycleAdapter) Updated(obj runtime.Object) (runtime.Object, error) {
	o, err := w.lifecycle.Updated(obj.(*ExecutionRun))
	if o == nil {
		return nil, err
	}
	return o, err
}

func NewExecutionRunLifecycleAdapter(name string, clusterScoped bool, client ExecutionRunInterface, l ExecutionRunLifecycle) ExecutionRunHandlerFunc {
	adapter := &executionRunLifecycleAdapter{lifecycle: l}
	syncFn := lifecycle.NewObjectLifecycleAdapter(name, clusterScoped, adapter, client.ObjectClient())
	return func(key string, obj *ExecutionRun) (runtime.Object, error) {
		newObj, err := syncFn(key, obj)
		if o, ok := newObj.(runtime.Object); ok {
			return o, err
		}
		return nil, err
	}
}
