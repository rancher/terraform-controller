package v1

import (
	"context"

	"github.com/rancher/norman/controller"
	"github.com/rancher/norman/objectclient"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

var (
	ExecutionRunGroupVersionKind = schema.GroupVersionKind{
		Version: Version,
		Group:   GroupName,
		Kind:    "ExecutionRun",
	}
	ExecutionRunResource = metav1.APIResource{
		Name:         "executionruns",
		SingularName: "executionrun",
		Namespaced:   true,

		Kind: ExecutionRunGroupVersionKind.Kind,
	}
)

type ExecutionRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExecutionRun
}

type ExecutionRunHandlerFunc func(key string, obj *ExecutionRun) (runtime.Object, error)

type ExecutionRunChangeHandlerFunc func(obj *ExecutionRun) (runtime.Object, error)

type ExecutionRunLister interface {
	List(namespace string, selector labels.Selector) (ret []*ExecutionRun, err error)
	Get(namespace, name string) (*ExecutionRun, error)
}

type ExecutionRunController interface {
	Generic() controller.GenericController
	Informer() cache.SharedIndexInformer
	Lister() ExecutionRunLister
	AddHandler(ctx context.Context, name string, handler ExecutionRunHandlerFunc)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, handler ExecutionRunHandlerFunc)
	Enqueue(namespace, name string)
	Sync(ctx context.Context) error
	Start(ctx context.Context, threadiness int) error
}

type ExecutionRunInterface interface {
	ObjectClient() *objectclient.ObjectClient
	Create(*ExecutionRun) (*ExecutionRun, error)
	GetNamespaced(namespace, name string, opts metav1.GetOptions) (*ExecutionRun, error)
	Get(name string, opts metav1.GetOptions) (*ExecutionRun, error)
	Update(*ExecutionRun) (*ExecutionRun, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (*ExecutionRunList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Controller() ExecutionRunController
	AddHandler(ctx context.Context, name string, sync ExecutionRunHandlerFunc)
	AddLifecycle(ctx context.Context, name string, lifecycle ExecutionRunLifecycle)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync ExecutionRunHandlerFunc)
	AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle ExecutionRunLifecycle)
}

type executionRunLister struct {
	controller *executionRunController
}

func (l *executionRunLister) List(namespace string, selector labels.Selector) (ret []*ExecutionRun, err error) {
	err = cache.ListAllByNamespace(l.controller.Informer().GetIndexer(), namespace, selector, func(obj interface{}) {
		ret = append(ret, obj.(*ExecutionRun))
	})
	return
}

func (l *executionRunLister) Get(namespace, name string) (*ExecutionRun, error) {
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
			Group:    ExecutionRunGroupVersionKind.Group,
			Resource: "executionRun",
		}, key)
	}
	return obj.(*ExecutionRun), nil
}

type executionRunController struct {
	controller.GenericController
}

func (c *executionRunController) Generic() controller.GenericController {
	return c.GenericController
}

func (c *executionRunController) Lister() ExecutionRunLister {
	return &executionRunLister{
		controller: c,
	}
}

func (c *executionRunController) AddHandler(ctx context.Context, name string, handler ExecutionRunHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*ExecutionRun); ok {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

func (c *executionRunController) AddClusterScopedHandler(ctx context.Context, name, cluster string, handler ExecutionRunHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*ExecutionRun); ok && controller.ObjectInCluster(cluster, obj) {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

type executionRunFactory struct {
}

func (c executionRunFactory) Object() runtime.Object {
	return &ExecutionRun{}
}

func (c executionRunFactory) List() runtime.Object {
	return &ExecutionRunList{}
}

func (s *executionRunClient) Controller() ExecutionRunController {
	s.client.Lock()
	defer s.client.Unlock()

	c, ok := s.client.executionRunControllers[s.ns]
	if ok {
		return c
	}

	genericController := controller.NewGenericController(ExecutionRunGroupVersionKind.Kind+"Controller",
		s.objectClient)

	c = &executionRunController{
		GenericController: genericController,
	}

	s.client.executionRunControllers[s.ns] = c
	s.client.starters = append(s.client.starters, c)

	return c
}

type executionRunClient struct {
	client       *Client
	ns           string
	objectClient *objectclient.ObjectClient
	controller   ExecutionRunController
}

func (s *executionRunClient) ObjectClient() *objectclient.ObjectClient {
	return s.objectClient
}

func (s *executionRunClient) Create(o *ExecutionRun) (*ExecutionRun, error) {
	obj, err := s.objectClient.Create(o)
	return obj.(*ExecutionRun), err
}

func (s *executionRunClient) Get(name string, opts metav1.GetOptions) (*ExecutionRun, error) {
	obj, err := s.objectClient.Get(name, opts)
	return obj.(*ExecutionRun), err
}

func (s *executionRunClient) GetNamespaced(namespace, name string, opts metav1.GetOptions) (*ExecutionRun, error) {
	obj, err := s.objectClient.GetNamespaced(namespace, name, opts)
	return obj.(*ExecutionRun), err
}

func (s *executionRunClient) Update(o *ExecutionRun) (*ExecutionRun, error) {
	obj, err := s.objectClient.Update(o.Name, o)
	return obj.(*ExecutionRun), err
}

func (s *executionRunClient) Delete(name string, options *metav1.DeleteOptions) error {
	return s.objectClient.Delete(name, options)
}

func (s *executionRunClient) DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error {
	return s.objectClient.DeleteNamespaced(namespace, name, options)
}

func (s *executionRunClient) List(opts metav1.ListOptions) (*ExecutionRunList, error) {
	obj, err := s.objectClient.List(opts)
	return obj.(*ExecutionRunList), err
}

func (s *executionRunClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return s.objectClient.Watch(opts)
}

// Patch applies the patch and returns the patched deployment.
func (s *executionRunClient) Patch(o *ExecutionRun, data []byte, subresources ...string) (*ExecutionRun, error) {
	obj, err := s.objectClient.Patch(o.Name, o, data, subresources...)
	return obj.(*ExecutionRun), err
}

func (s *executionRunClient) DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return s.objectClient.DeleteCollection(deleteOpts, listOpts)
}

func (s *executionRunClient) AddHandler(ctx context.Context, name string, sync ExecutionRunHandlerFunc) {
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *executionRunClient) AddLifecycle(ctx context.Context, name string, lifecycle ExecutionRunLifecycle) {
	sync := NewExecutionRunLifecycleAdapter(name, false, s, lifecycle)
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *executionRunClient) AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync ExecutionRunHandlerFunc) {
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}

func (s *executionRunClient) AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle ExecutionRunLifecycle) {
	sync := NewExecutionRunLifecycleAdapter(name+"_"+clusterName, true, s, lifecycle)
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}

type ExecutionRunIndexer func(obj *ExecutionRun) ([]string, error)

type ExecutionRunClientCache interface {
	Get(namespace, name string) (*ExecutionRun, error)
	List(namespace string, selector labels.Selector) ([]*ExecutionRun, error)

	Index(name string, indexer ExecutionRunIndexer)
	GetIndexed(name, key string) ([]*ExecutionRun, error)
}

type ExecutionRunClient interface {
	Create(*ExecutionRun) (*ExecutionRun, error)
	Get(namespace, name string, opts metav1.GetOptions) (*ExecutionRun, error)
	Update(*ExecutionRun) (*ExecutionRun, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	List(namespace string, opts metav1.ListOptions) (*ExecutionRunList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)

	Cache() ExecutionRunClientCache

	OnCreate(ctx context.Context, name string, sync ExecutionRunChangeHandlerFunc)
	OnChange(ctx context.Context, name string, sync ExecutionRunChangeHandlerFunc)
	OnRemove(ctx context.Context, name string, sync ExecutionRunChangeHandlerFunc)
	Enqueue(namespace, name string)

	Generic() controller.GenericController
	Interface() ExecutionRunInterface
}

type executionRunClientCache struct {
	client *executionRunClient2
}

type executionRunClient2 struct {
	iface      ExecutionRunInterface
	controller ExecutionRunController
}

func (n *executionRunClient2) Interface() ExecutionRunInterface {
	return n.iface
}

func (n *executionRunClient2) Generic() controller.GenericController {
	return n.iface.Controller().Generic()
}

func (n *executionRunClient2) Enqueue(namespace, name string) {
	n.iface.Controller().Enqueue(namespace, name)
}

func (n *executionRunClient2) Create(obj *ExecutionRun) (*ExecutionRun, error) {
	return n.iface.Create(obj)
}

func (n *executionRunClient2) Get(namespace, name string, opts metav1.GetOptions) (*ExecutionRun, error) {
	return n.iface.GetNamespaced(namespace, name, opts)
}

func (n *executionRunClient2) Update(obj *ExecutionRun) (*ExecutionRun, error) {
	return n.iface.Update(obj)
}

func (n *executionRunClient2) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return n.iface.DeleteNamespaced(namespace, name, options)
}

func (n *executionRunClient2) List(namespace string, opts metav1.ListOptions) (*ExecutionRunList, error) {
	return n.iface.List(opts)
}

func (n *executionRunClient2) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return n.iface.Watch(opts)
}

func (n *executionRunClientCache) Get(namespace, name string) (*ExecutionRun, error) {
	return n.client.controller.Lister().Get(namespace, name)
}

func (n *executionRunClientCache) List(namespace string, selector labels.Selector) ([]*ExecutionRun, error) {
	return n.client.controller.Lister().List(namespace, selector)
}

func (n *executionRunClient2) Cache() ExecutionRunClientCache {
	n.loadController()
	return &executionRunClientCache{
		client: n,
	}
}

func (n *executionRunClient2) OnCreate(ctx context.Context, name string, sync ExecutionRunChangeHandlerFunc) {
	n.loadController()
	n.iface.AddLifecycle(ctx, name+"-create", &executionRunLifecycleDelegate{create: sync})
}

func (n *executionRunClient2) OnChange(ctx context.Context, name string, sync ExecutionRunChangeHandlerFunc) {
	n.loadController()
	n.iface.AddLifecycle(ctx, name+"-change", &executionRunLifecycleDelegate{update: sync})
}

func (n *executionRunClient2) OnRemove(ctx context.Context, name string, sync ExecutionRunChangeHandlerFunc) {
	n.loadController()
	n.iface.AddLifecycle(ctx, name, &executionRunLifecycleDelegate{remove: sync})
}

func (n *executionRunClientCache) Index(name string, indexer ExecutionRunIndexer) {
	err := n.client.controller.Informer().GetIndexer().AddIndexers(map[string]cache.IndexFunc{
		name: func(obj interface{}) ([]string, error) {
			if v, ok := obj.(*ExecutionRun); ok {
				return indexer(v)
			}
			return nil, nil
		},
	})

	if err != nil {
		panic(err)
	}
}

func (n *executionRunClientCache) GetIndexed(name, key string) ([]*ExecutionRun, error) {
	var result []*ExecutionRun
	objs, err := n.client.controller.Informer().GetIndexer().ByIndex(name, key)
	if err != nil {
		return nil, err
	}
	for _, obj := range objs {
		if v, ok := obj.(*ExecutionRun); ok {
			result = append(result, v)
		}
	}

	return result, nil
}

func (n *executionRunClient2) loadController() {
	if n.controller == nil {
		n.controller = n.iface.Controller()
	}
}

type executionRunLifecycleDelegate struct {
	create ExecutionRunChangeHandlerFunc
	update ExecutionRunChangeHandlerFunc
	remove ExecutionRunChangeHandlerFunc
}

func (n *executionRunLifecycleDelegate) HasCreate() bool {
	return n.create != nil
}

func (n *executionRunLifecycleDelegate) Create(obj *ExecutionRun) (runtime.Object, error) {
	if n.create == nil {
		return obj, nil
	}
	return n.create(obj)
}

func (n *executionRunLifecycleDelegate) HasFinalize() bool {
	return n.remove != nil
}

func (n *executionRunLifecycleDelegate) Remove(obj *ExecutionRun) (runtime.Object, error) {
	if n.remove == nil {
		return obj, nil
	}
	return n.remove(obj)
}

func (n *executionRunLifecycleDelegate) Updated(obj *ExecutionRun) (runtime.Object, error) {
	if n.update == nil {
		return obj, nil
	}
	return n.update(obj)
}
