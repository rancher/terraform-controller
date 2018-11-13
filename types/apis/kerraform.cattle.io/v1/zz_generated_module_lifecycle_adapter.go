package v1

import (
	"github.com/rancher/norman/lifecycle"
	"k8s.io/apimachinery/pkg/runtime"
)

type ModuleLifecycle interface {
	Create(obj *Module) (runtime.Object, error)
	Remove(obj *Module) (runtime.Object, error)
	Updated(obj *Module) (runtime.Object, error)
}

type moduleLifecycleAdapter struct {
	lifecycle ModuleLifecycle
}

func (w *moduleLifecycleAdapter) Create(obj runtime.Object) (runtime.Object, error) {
	o, err := w.lifecycle.Create(obj.(*Module))
	if o == nil {
		return nil, err
	}
	return o, err
}

func (w *moduleLifecycleAdapter) Finalize(obj runtime.Object) (runtime.Object, error) {
	o, err := w.lifecycle.Remove(obj.(*Module))
	if o == nil {
		return nil, err
	}
	return o, err
}

func (w *moduleLifecycleAdapter) Updated(obj runtime.Object) (runtime.Object, error) {
	o, err := w.lifecycle.Updated(obj.(*Module))
	if o == nil {
		return nil, err
	}
	return o, err
}

func NewModuleLifecycleAdapter(name string, clusterScoped bool, client ModuleInterface, l ModuleLifecycle) ModuleHandlerFunc {
	adapter := &moduleLifecycleAdapter{lifecycle: l}
	syncFn := lifecycle.NewObjectLifecycleAdapter(name, clusterScoped, adapter, client.ObjectClient())
	return func(key string, obj *Module) (runtime.Object, error) {
		newObj, err := syncFn(key, obj)
		if o, ok := newObj.(runtime.Object); ok {
			return o, err
		}
		return nil, err
	}
}
