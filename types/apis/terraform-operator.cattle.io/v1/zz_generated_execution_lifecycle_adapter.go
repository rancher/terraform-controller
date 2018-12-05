package v1

import (
	"github.com/rancher/norman/lifecycle"
	"k8s.io/apimachinery/pkg/runtime"
)

type ExecutionLifecycle interface {
	Create(obj *Execution) (runtime.Object, error)
	Remove(obj *Execution) (runtime.Object, error)
	Updated(obj *Execution) (runtime.Object, error)
}

type executionLifecycleAdapter struct {
	lifecycle ExecutionLifecycle
}

func (w *executionLifecycleAdapter) HasCreate() bool {
	o, ok := w.lifecycle.(lifecycle.ObjectLifecycleCondition)
	return !ok || o.HasCreate()
}

func (w *executionLifecycleAdapter) HasFinalize() bool {
	o, ok := w.lifecycle.(lifecycle.ObjectLifecycleCondition)
	return !ok || o.HasFinalize()
}

func (w *executionLifecycleAdapter) Create(obj runtime.Object) (runtime.Object, error) {
	o, err := w.lifecycle.Create(obj.(*Execution))
	if o == nil {
		return nil, err
	}
	return o, err
}

func (w *executionLifecycleAdapter) Finalize(obj runtime.Object) (runtime.Object, error) {
	o, err := w.lifecycle.Remove(obj.(*Execution))
	if o == nil {
		return nil, err
	}
	return o, err
}

func (w *executionLifecycleAdapter) Updated(obj runtime.Object) (runtime.Object, error) {
	o, err := w.lifecycle.Updated(obj.(*Execution))
	if o == nil {
		return nil, err
	}
	return o, err
}

func NewExecutionLifecycleAdapter(name string, clusterScoped bool, client ExecutionInterface, l ExecutionLifecycle) ExecutionHandlerFunc {
	adapter := &executionLifecycleAdapter{lifecycle: l}
	syncFn := lifecycle.NewObjectLifecycleAdapter(name, clusterScoped, adapter, client.ObjectClient())
	return func(key string, obj *Execution) (runtime.Object, error) {
		newObj, err := syncFn(key, obj)
		if o, ok := newObj.(runtime.Object); ok {
			return o, err
		}
		return nil, err
	}
}
