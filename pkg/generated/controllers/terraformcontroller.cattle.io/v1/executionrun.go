/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by main. DO NOT EDIT.

package v1

import (
	"context"

	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	clientset "github.com/rancher/terraform-controller/pkg/generated/clientset/versioned/typed/terraformcontroller.cattle.io/v1"
	informers "github.com/rancher/terraform-controller/pkg/generated/informers/externalversions/terraformcontroller.cattle.io/v1"
	listers "github.com/rancher/terraform-controller/pkg/generated/listers/terraformcontroller.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/generic"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type ExecutionRunHandler func(string, *v1.ExecutionRun) (*v1.ExecutionRun, error)

type ExecutionRunController interface {
	ExecutionRunClient

	OnChange(ctx context.Context, name string, sync ExecutionRunHandler)
	OnRemove(ctx context.Context, name string, sync ExecutionRunHandler)
	Enqueue(namespace, name string)

	Cache() ExecutionRunCache

	Informer() cache.SharedIndexInformer
	GroupVersionKind() schema.GroupVersionKind

	AddGenericHandler(ctx context.Context, name string, handler generic.Handler)
	AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler)
	Updater() generic.Updater
}

type ExecutionRunClient interface {
	Create(*v1.ExecutionRun) (*v1.ExecutionRun, error)
	Update(*v1.ExecutionRun) (*v1.ExecutionRun, error)
	UpdateStatus(*v1.ExecutionRun) (*v1.ExecutionRun, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	Get(namespace, name string, options metav1.GetOptions) (*v1.ExecutionRun, error)
	List(namespace string, opts metav1.ListOptions) (*v1.ExecutionRunList, error)
	Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error)
	Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.ExecutionRun, err error)
}

type ExecutionRunCache interface {
	Get(namespace, name string) (*v1.ExecutionRun, error)
	List(namespace string, selector labels.Selector) ([]*v1.ExecutionRun, error)

	AddIndexer(indexName string, indexer ExecutionRunIndexer)
	GetByIndex(indexName, key string) ([]*v1.ExecutionRun, error)
}

type ExecutionRunIndexer func(obj *v1.ExecutionRun) ([]string, error)

type executionRunController struct {
	controllerManager *generic.ControllerManager
	clientGetter      clientset.ExecutionRunsGetter
	informer          informers.ExecutionRunInformer
	gvk               schema.GroupVersionKind
}

func NewExecutionRunController(gvk schema.GroupVersionKind, controllerManager *generic.ControllerManager, clientGetter clientset.ExecutionRunsGetter, informer informers.ExecutionRunInformer) ExecutionRunController {
	return &executionRunController{
		controllerManager: controllerManager,
		clientGetter:      clientGetter,
		informer:          informer,
		gvk:               gvk,
	}
}

func FromExecutionRunHandlerToHandler(sync ExecutionRunHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v1.ExecutionRun
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v1.ExecutionRun))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *executionRunController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v1.ExecutionRun))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateExecutionRunOnChange(updater generic.Updater, handler ExecutionRunHandler) ExecutionRunHandler {
	return func(key string, obj *v1.ExecutionRun) (*v1.ExecutionRun, error) {
		if obj == nil {
			return handler(key, nil)
		}

		copyObj := obj.DeepCopy()
		newObj, err := handler(key, copyObj)
		if newObj != nil {
			copyObj = newObj
		}
		if obj.ResourceVersion == copyObj.ResourceVersion && !equality.Semantic.DeepEqual(obj, copyObj) {
			newObj, err := updater(copyObj)
			if newObj != nil && err == nil {
				copyObj = newObj.(*v1.ExecutionRun)
			}
		}

		return copyObj, err
	}
}

func (c *executionRunController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controllerManager.AddHandler(ctx, c.gvk, c.informer.Informer(), name, handler)
}

func (c *executionRunController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	removeHandler := generic.NewRemoveHandler(name, c.Updater(), handler)
	c.controllerManager.AddHandler(ctx, c.gvk, c.informer.Informer(), name, removeHandler)
}

func (c *executionRunController) OnChange(ctx context.Context, name string, sync ExecutionRunHandler) {
	c.AddGenericHandler(ctx, name, FromExecutionRunHandlerToHandler(sync))
}

func (c *executionRunController) OnRemove(ctx context.Context, name string, sync ExecutionRunHandler) {
	removeHandler := generic.NewRemoveHandler(name, c.Updater(), FromExecutionRunHandlerToHandler(sync))
	c.AddGenericHandler(ctx, name, removeHandler)
}

func (c *executionRunController) Enqueue(namespace, name string) {
	c.controllerManager.Enqueue(c.gvk, namespace, name)
}

func (c *executionRunController) Informer() cache.SharedIndexInformer {
	return c.informer.Informer()
}

func (c *executionRunController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *executionRunController) Cache() ExecutionRunCache {
	return &executionRunCache{
		lister:  c.informer.Lister(),
		indexer: c.informer.Informer().GetIndexer(),
	}
}

func (c *executionRunController) Create(obj *v1.ExecutionRun) (*v1.ExecutionRun, error) {
	return c.clientGetter.ExecutionRuns(obj.Namespace).Create(obj)
}

func (c *executionRunController) Update(obj *v1.ExecutionRun) (*v1.ExecutionRun, error) {
	return c.clientGetter.ExecutionRuns(obj.Namespace).Update(obj)
}

func (c *executionRunController) UpdateStatus(obj *v1.ExecutionRun) (*v1.ExecutionRun, error) {
	return c.clientGetter.ExecutionRuns(obj.Namespace).UpdateStatus(obj)
}

func (c *executionRunController) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return c.clientGetter.ExecutionRuns(namespace).Delete(name, options)
}

func (c *executionRunController) Get(namespace, name string, options metav1.GetOptions) (*v1.ExecutionRun, error) {
	return c.clientGetter.ExecutionRuns(namespace).Get(name, options)
}

func (c *executionRunController) List(namespace string, opts metav1.ListOptions) (*v1.ExecutionRunList, error) {
	return c.clientGetter.ExecutionRuns(namespace).List(opts)
}

func (c *executionRunController) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.clientGetter.ExecutionRuns(namespace).Watch(opts)
}

func (c *executionRunController) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.ExecutionRun, err error) {
	return c.clientGetter.ExecutionRuns(namespace).Patch(name, pt, data, subresources...)
}

type executionRunCache struct {
	lister  listers.ExecutionRunLister
	indexer cache.Indexer
}

func (c *executionRunCache) Get(namespace, name string) (*v1.ExecutionRun, error) {
	return c.lister.ExecutionRuns(namespace).Get(name)
}

func (c *executionRunCache) List(namespace string, selector labels.Selector) ([]*v1.ExecutionRun, error) {
	return c.lister.ExecutionRuns(namespace).List(selector)
}

func (c *executionRunCache) AddIndexer(indexName string, indexer ExecutionRunIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v1.ExecutionRun))
		},
	}))
}

func (c *executionRunCache) GetByIndex(indexName, key string) (result []*v1.ExecutionRun, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	for _, obj := range objs {
		result = append(result, obj.(*v1.ExecutionRun))
	}
	return result, nil
}
