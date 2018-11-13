package v1

import (
	"context"

	"github.com/rancher/norman/controller"
	"github.com/rancher/norman/objectclient"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

var (
	ConfigMapGroupVersionKind = schema.GroupVersionKind{
		Version: Version,
		Group:   GroupName,
		Kind:    "ConfigMap",
	}
	ConfigMapResource = metav1.APIResource{
		Name:         "configmaps",
		SingularName: "configmap",
		Namespaced:   true,

		Kind: ConfigMapGroupVersionKind.Kind,
	}
)

type ConfigMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []v1.ConfigMap
}

type ConfigMapHandlerFunc func(key string, obj *v1.ConfigMap) (runtime.Object, error)

type ConfigMapChangeHandlerFunc func(obj *v1.ConfigMap) (runtime.Object, error)

type ConfigMapLister interface {
	List(namespace string, selector labels.Selector) (ret []*v1.ConfigMap, err error)
	Get(namespace, name string) (*v1.ConfigMap, error)
}

type ConfigMapController interface {
	Generic() controller.GenericController
	Informer() cache.SharedIndexInformer
	Lister() ConfigMapLister
	AddHandler(ctx context.Context, name string, handler ConfigMapHandlerFunc)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, handler ConfigMapHandlerFunc)
	Enqueue(namespace, name string)
	Sync(ctx context.Context) error
	Start(ctx context.Context, threadiness int) error
}

type ConfigMapInterface interface {
	ObjectClient() *objectclient.ObjectClient
	Create(*v1.ConfigMap) (*v1.ConfigMap, error)
	GetNamespaced(namespace, name string, opts metav1.GetOptions) (*v1.ConfigMap, error)
	Get(name string, opts metav1.GetOptions) (*v1.ConfigMap, error)
	Update(*v1.ConfigMap) (*v1.ConfigMap, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (*v1.ConfigMapList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Controller() ConfigMapController
	AddHandler(ctx context.Context, name string, sync ConfigMapHandlerFunc)
	AddLifecycle(ctx context.Context, name string, lifecycle ConfigMapLifecycle)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync ConfigMapHandlerFunc)
	AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle ConfigMapLifecycle)
}

type configMapLister struct {
	controller *configMapController
}

func (l *configMapLister) List(namespace string, selector labels.Selector) (ret []*v1.ConfigMap, err error) {
	err = cache.ListAllByNamespace(l.controller.Informer().GetIndexer(), namespace, selector, func(obj interface{}) {
		ret = append(ret, obj.(*v1.ConfigMap))
	})
	return
}

func (l *configMapLister) Get(namespace, name string) (*v1.ConfigMap, error) {
	var key string
	if namespace != "" {
		key = namespace + "/" + name
	} else {
		key = name
	}
	obj, exists, err := l.controller.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    ConfigMapGroupVersionKind.Group,
			Resource: "configMap",
		}, key)
	}
	return obj.(*v1.ConfigMap), nil
}

type configMapController struct {
	controller.GenericController
}

func (c *configMapController) Generic() controller.GenericController {
	return c.GenericController
}

func (c *configMapController) Lister() ConfigMapLister {
	return &configMapLister{
		controller: c,
	}
}

func (c *configMapController) AddHandler(ctx context.Context, name string, handler ConfigMapHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*v1.ConfigMap); ok {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

func (c *configMapController) AddClusterScopedHandler(ctx context.Context, name, cluster string, handler ConfigMapHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*v1.ConfigMap); ok && controller.ObjectInCluster(cluster, obj) {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

type configMapFactory struct {
}

func (c configMapFactory) Object() runtime.Object {
	return &v1.ConfigMap{}
}

func (c configMapFactory) List() runtime.Object {
	return &ConfigMapList{}
}

func (s *configMapClient) Controller() ConfigMapController {
	s.client.Lock()
	defer s.client.Unlock()

	c, ok := s.client.configMapControllers[s.ns]
	if ok {
		return c
	}

	genericController := controller.NewGenericController(ConfigMapGroupVersionKind.Kind+"Controller",
		s.objectClient)

	c = &configMapController{
		GenericController: genericController,
	}

	s.client.configMapControllers[s.ns] = c
	s.client.starters = append(s.client.starters, c)

	return c
}

type configMapClient struct {
	client       *Client
	ns           string
	objectClient *objectclient.ObjectClient
	controller   ConfigMapController
}

func (s *configMapClient) ObjectClient() *objectclient.ObjectClient {
	return s.objectClient
}

func (s *configMapClient) Create(o *v1.ConfigMap) (*v1.ConfigMap, error) {
	obj, err := s.objectClient.Create(o)
	return obj.(*v1.ConfigMap), err
}

func (s *configMapClient) Get(name string, opts metav1.GetOptions) (*v1.ConfigMap, error) {
	obj, err := s.objectClient.Get(name, opts)
	return obj.(*v1.ConfigMap), err
}

func (s *configMapClient) GetNamespaced(namespace, name string, opts metav1.GetOptions) (*v1.ConfigMap, error) {
	obj, err := s.objectClient.GetNamespaced(namespace, name, opts)
	return obj.(*v1.ConfigMap), err
}

func (s *configMapClient) Update(o *v1.ConfigMap) (*v1.ConfigMap, error) {
	obj, err := s.objectClient.Update(o.Name, o)
	return obj.(*v1.ConfigMap), err
}

func (s *configMapClient) Delete(name string, options *metav1.DeleteOptions) error {
	return s.objectClient.Delete(name, options)
}

func (s *configMapClient) DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error {
	return s.objectClient.DeleteNamespaced(namespace, name, options)
}

func (s *configMapClient) List(opts metav1.ListOptions) (*v1.ConfigMapList, error) {
	obj, err := s.objectClient.List(opts)
	return obj.(*v1.ConfigMapList), err
}

func (s *configMapClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return s.objectClient.Watch(opts)
}

// Patch applies the patch and returns the patched deployment.
func (s *configMapClient) Patch(o *v1.ConfigMap, data []byte, subresources ...string) (*v1.ConfigMap, error) {
	obj, err := s.objectClient.Patch(o.Name, o, data, subresources...)
	return obj.(*v1.ConfigMap), err
}

func (s *configMapClient) DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return s.objectClient.DeleteCollection(deleteOpts, listOpts)
}

func (s *configMapClient) AddHandler(ctx context.Context, name string, sync ConfigMapHandlerFunc) {
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *configMapClient) AddLifecycle(ctx context.Context, name string, lifecycle ConfigMapLifecycle) {
	sync := NewConfigMapLifecycleAdapter(name, false, s, lifecycle)
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *configMapClient) AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync ConfigMapHandlerFunc) {
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}

func (s *configMapClient) AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle ConfigMapLifecycle) {
	sync := NewConfigMapLifecycleAdapter(name+"_"+clusterName, true, s, lifecycle)
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}

type ConfigMapIndexer func(obj *v1.ConfigMap) ([]string, error)

type ConfigMapClient interface {
	Interface() ConfigMapInterface
	Create(*v1.ConfigMap) (*v1.ConfigMap, error)
	Get(namespace, name string, opts metav1.GetOptions) (*v1.ConfigMap, error)
	Update(*v1.ConfigMap) (*v1.ConfigMap, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	List(namespace string, opts metav1.ListOptions) (*v1.ConfigMapList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)

	GetCached(namespace, name string) (*v1.ConfigMap, error)
	ListCached(namespace string, selector labels.Selector) ([]*v1.ConfigMap, error)
	EnableCache()

	OnCreate(ctx context.Context, name string, sync ConfigMapChangeHandlerFunc)
	OnChange(ctx context.Context, name string, sync ConfigMapChangeHandlerFunc)
	OnRemove(ctx context.Context, name string, sync ConfigMapChangeHandlerFunc)

	Index(name string, indexer ConfigMapIndexer)
	GetIndexed(name, key string) ([]*v1.ConfigMap, error)
}

type configMapClient2 struct {
	iface      ConfigMapInterface
	controller ConfigMapController
}

func (n *configMapClient2) Interface() ConfigMapInterface {
	return n.iface
}

func (n *configMapClient2) Create(obj *v1.ConfigMap) (*v1.ConfigMap, error) {
	return n.iface.Create(obj)
}

func (n *configMapClient2) Get(namespace, name string, opts metav1.GetOptions) (*v1.ConfigMap, error) {
	return n.iface.GetNamespaced(namespace, name, opts)
}

func (n *configMapClient2) Update(obj *v1.ConfigMap) (*v1.ConfigMap, error) {
	return n.iface.Update(obj)
}

func (n *configMapClient2) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return n.iface.DeleteNamespaced(namespace, name, options)
}

func (n *configMapClient2) List(namespace string, opts metav1.ListOptions) (*v1.ConfigMapList, error) {
	return n.iface.List(opts)
}

func (n *configMapClient2) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return n.iface.Watch(opts)
}

func (n *configMapClient2) GetCached(namespace, name string) (*v1.ConfigMap, error) {
	n.assertCache()
	return n.controller.Lister().Get(namespace, name)
}

func (n *configMapClient2) ListCached(namespace string, selector labels.Selector) ([]*v1.ConfigMap, error) {
	n.assertCache()
	return n.controller.Lister().List(namespace, selector)
}

func (n *configMapClient2) EnableCache() {
	n.loadController()
}

func (n *configMapClient2) OnCreate(ctx context.Context, name string, sync ConfigMapChangeHandlerFunc) {
	n.loadController()
	n.iface.AddLifecycle(ctx, name, &configMapLifecycleDelegate{create: sync})
}

func (n *configMapClient2) OnChange(ctx context.Context, name string, sync ConfigMapChangeHandlerFunc) {
	n.loadController()
	n.iface.AddLifecycle(ctx, name, &configMapLifecycleDelegate{update: sync})
}

func (n *configMapClient2) OnRemove(ctx context.Context, name string, sync ConfigMapChangeHandlerFunc) {
	n.loadController()
	n.iface.AddLifecycle(ctx, name, &configMapLifecycleDelegate{remove: sync})
}

func (n *configMapClient2) Index(name string, indexer ConfigMapIndexer) {
	n.loadController()
	err := n.controller.Informer().GetIndexer().AddIndexers(map[string]cache.IndexFunc{
		name: func(obj interface{}) ([]string, error) {
			if v, ok := obj.(*v1.ConfigMap); ok {
				return indexer(v)
			}
			return nil, nil
		},
	})

	if err != nil {
		panic(err)
	}
}

func (n *configMapClient2) GetIndexed(name, key string) ([]*v1.ConfigMap, error) {
	n.assertCache()
	var result []*v1.ConfigMap
	objs, err := n.controller.Informer().GetIndexer().ByIndex(name, key)
	if err != nil {
		return nil, err
	}
	for _, obj := range objs {
		if v, ok := obj.(*v1.ConfigMap); ok {
			result = append(result, v)
		}
	}

	return result, nil
}

func (n *configMapClient2) loadController() {
	if n.controller == nil {
		n.controller = n.iface.Controller()
	}
}

func (n *configMapClient2) assertCache() {
	if n.controller == nil {
		panic("ConfigMap cache is not enabled, enable with ConfigMapClient.EnableCache()")
	}
}

type configMapLifecycleDelegate struct {
	create ConfigMapChangeHandlerFunc
	update ConfigMapChangeHandlerFunc
	remove ConfigMapChangeHandlerFunc
}

func (n *configMapLifecycleDelegate) Create(obj *v1.ConfigMap) (runtime.Object, error) {
	if n.create == nil {
		return obj, nil
	}
	return n.create(obj)
}

func (n *configMapLifecycleDelegate) Remove(obj *v1.ConfigMap) (runtime.Object, error) {
	if n.remove == nil {
		return obj, nil
	}
	return n.remove(obj)
}

func (n *configMapLifecycleDelegate) Updated(obj *v1.ConfigMap) (runtime.Object, error) {
	if n.update == nil {
		return obj, nil
	}
	return n.update(obj)
}
