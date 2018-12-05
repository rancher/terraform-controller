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
	ExecutionGroupVersionKind = schema.GroupVersionKind{
		Version: Version,
		Group:   GroupName,
		Kind:    "Execution",
	}
	ExecutionResource = metav1.APIResource{
		Name:         "executions",
		SingularName: "execution",
		Namespaced:   true,

		Kind: ExecutionGroupVersionKind.Kind,
	}
)

type ExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Execution
}

type ExecutionHandlerFunc func(key string, obj *Execution) (runtime.Object, error)

type ExecutionChangeHandlerFunc func(obj *Execution) (runtime.Object, error)

type ExecutionLister interface {
	List(namespace string, selector labels.Selector) (ret []*Execution, err error)
	Get(namespace, name string) (*Execution, error)
}

type ExecutionController interface {
	Generic() controller.GenericController
	Informer() cache.SharedIndexInformer
	Lister() ExecutionLister
	AddHandler(ctx context.Context, name string, handler ExecutionHandlerFunc)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, handler ExecutionHandlerFunc)
	Enqueue(namespace, name string)
	Sync(ctx context.Context) error
	Start(ctx context.Context, threadiness int) error
}

type ExecutionInterface interface {
	ObjectClient() *objectclient.ObjectClient
	Create(*Execution) (*Execution, error)
	GetNamespaced(namespace, name string, opts metav1.GetOptions) (*Execution, error)
	Get(name string, opts metav1.GetOptions) (*Execution, error)
	Update(*Execution) (*Execution, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (*ExecutionList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Controller() ExecutionController
	AddHandler(ctx context.Context, name string, sync ExecutionHandlerFunc)
	AddLifecycle(ctx context.Context, name string, lifecycle ExecutionLifecycle)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync ExecutionHandlerFunc)
	AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle ExecutionLifecycle)
}

type executionLister struct {
	controller *executionController
}

func (l *executionLister) List(namespace string, selector labels.Selector) (ret []*Execution, err error) {
	err = cache.ListAllByNamespace(l.controller.Informer().GetIndexer(), namespace, selector, func(obj interface{}) {
		ret = append(ret, obj.(*Execution))
	})
	return
}

func (l *executionLister) Get(namespace, name string) (*Execution, error) {
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
			Group:    ExecutionGroupVersionKind.Group,
			Resource: "execution",
		}, key)
	}
	return obj.(*Execution), nil
}

type executionController struct {
	controller.GenericController
}

func (c *executionController) Generic() controller.GenericController {
	return c.GenericController
}

func (c *executionController) Lister() ExecutionLister {
	return &executionLister{
		controller: c,
	}
}

func (c *executionController) AddHandler(ctx context.Context, name string, handler ExecutionHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*Execution); ok {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

func (c *executionController) AddClusterScopedHandler(ctx context.Context, name, cluster string, handler ExecutionHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*Execution); ok && controller.ObjectInCluster(cluster, obj) {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

type executionFactory struct {
}

func (c executionFactory) Object() runtime.Object {
	return &Execution{}
}

func (c executionFactory) List() runtime.Object {
	return &ExecutionList{}
}

func (s *executionClient) Controller() ExecutionController {
	s.client.Lock()
	defer s.client.Unlock()

	c, ok := s.client.executionControllers[s.ns]
	if ok {
		return c
	}

	genericController := controller.NewGenericController(ExecutionGroupVersionKind.Kind+"Controller",
		s.objectClient)

	c = &executionController{
		GenericController: genericController,
	}

	s.client.executionControllers[s.ns] = c
	s.client.starters = append(s.client.starters, c)

	return c
}

type executionClient struct {
	client       *Client
	ns           string
	objectClient *objectclient.ObjectClient
	controller   ExecutionController
}

func (s *executionClient) ObjectClient() *objectclient.ObjectClient {
	return s.objectClient
}

func (s *executionClient) Create(o *Execution) (*Execution, error) {
	obj, err := s.objectClient.Create(o)
	return obj.(*Execution), err
}

func (s *executionClient) Get(name string, opts metav1.GetOptions) (*Execution, error) {
	obj, err := s.objectClient.Get(name, opts)
	return obj.(*Execution), err
}

func (s *executionClient) GetNamespaced(namespace, name string, opts metav1.GetOptions) (*Execution, error) {
	obj, err := s.objectClient.GetNamespaced(namespace, name, opts)
	return obj.(*Execution), err
}

func (s *executionClient) Update(o *Execution) (*Execution, error) {
	obj, err := s.objectClient.Update(o.Name, o)
	return obj.(*Execution), err
}

func (s *executionClient) Delete(name string, options *metav1.DeleteOptions) error {
	return s.objectClient.Delete(name, options)
}

func (s *executionClient) DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error {
	return s.objectClient.DeleteNamespaced(namespace, name, options)
}

func (s *executionClient) List(opts metav1.ListOptions) (*ExecutionList, error) {
	obj, err := s.objectClient.List(opts)
	return obj.(*ExecutionList), err
}

func (s *executionClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return s.objectClient.Watch(opts)
}

// Patch applies the patch and returns the patched deployment.
func (s *executionClient) Patch(o *Execution, data []byte, subresources ...string) (*Execution, error) {
	obj, err := s.objectClient.Patch(o.Name, o, data, subresources...)
	return obj.(*Execution), err
}

func (s *executionClient) DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return s.objectClient.DeleteCollection(deleteOpts, listOpts)
}

func (s *executionClient) AddHandler(ctx context.Context, name string, sync ExecutionHandlerFunc) {
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *executionClient) AddLifecycle(ctx context.Context, name string, lifecycle ExecutionLifecycle) {
	sync := NewExecutionLifecycleAdapter(name, false, s, lifecycle)
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *executionClient) AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync ExecutionHandlerFunc) {
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}

func (s *executionClient) AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle ExecutionLifecycle) {
	sync := NewExecutionLifecycleAdapter(name+"_"+clusterName, true, s, lifecycle)
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}

type ExecutionIndexer func(obj *Execution) ([]string, error)

type ExecutionClientCache interface {
	Get(namespace, name string) (*Execution, error)
	List(namespace string, selector labels.Selector) ([]*Execution, error)

	Index(name string, indexer ExecutionIndexer)
	GetIndexed(name, key string) ([]*Execution, error)
}

type ExecutionClient interface {
	Create(*Execution) (*Execution, error)
	Get(namespace, name string, opts metav1.GetOptions) (*Execution, error)
	Update(*Execution) (*Execution, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	List(namespace string, opts metav1.ListOptions) (*ExecutionList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)

	Cache() ExecutionClientCache

	OnCreate(ctx context.Context, name string, sync ExecutionChangeHandlerFunc)
	OnChange(ctx context.Context, name string, sync ExecutionChangeHandlerFunc)
	OnRemove(ctx context.Context, name string, sync ExecutionChangeHandlerFunc)
	Enqueue(namespace, name string)

	Generic() controller.GenericController
	Interface() ExecutionInterface
}

type executionClientCache struct {
	client *executionClient2
}

type executionClient2 struct {
	iface      ExecutionInterface
	controller ExecutionController
}

func (n *executionClient2) Interface() ExecutionInterface {
	return n.iface
}

func (n *executionClient2) Generic() controller.GenericController {
	return n.iface.Controller().Generic()
}

func (n *executionClient2) Enqueue(namespace, name string) {
	n.iface.Controller().Enqueue(namespace, name)
}

func (n *executionClient2) Create(obj *Execution) (*Execution, error) {
	return n.iface.Create(obj)
}

func (n *executionClient2) Get(namespace, name string, opts metav1.GetOptions) (*Execution, error) {
	return n.iface.GetNamespaced(namespace, name, opts)
}

func (n *executionClient2) Update(obj *Execution) (*Execution, error) {
	return n.iface.Update(obj)
}

func (n *executionClient2) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return n.iface.DeleteNamespaced(namespace, name, options)
}

func (n *executionClient2) List(namespace string, opts metav1.ListOptions) (*ExecutionList, error) {
	return n.iface.List(opts)
}

func (n *executionClient2) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return n.iface.Watch(opts)
}

func (n *executionClientCache) Get(namespace, name string) (*Execution, error) {
	return n.client.controller.Lister().Get(namespace, name)
}

func (n *executionClientCache) List(namespace string, selector labels.Selector) ([]*Execution, error) {
	return n.client.controller.Lister().List(namespace, selector)
}

func (n *executionClient2) Cache() ExecutionClientCache {
	n.loadController()
	return &executionClientCache{
		client: n,
	}
}

func (n *executionClient2) OnCreate(ctx context.Context, name string, sync ExecutionChangeHandlerFunc) {
	n.loadController()
	n.iface.AddLifecycle(ctx, name+"-create", &executionLifecycleDelegate{create: sync})
}

func (n *executionClient2) OnChange(ctx context.Context, name string, sync ExecutionChangeHandlerFunc) {
	n.loadController()
	n.iface.AddLifecycle(ctx, name+"-change", &executionLifecycleDelegate{update: sync})
}

func (n *executionClient2) OnRemove(ctx context.Context, name string, sync ExecutionChangeHandlerFunc) {
	n.loadController()
	n.iface.AddLifecycle(ctx, name, &executionLifecycleDelegate{remove: sync})
}

func (n *executionClientCache) Index(name string, indexer ExecutionIndexer) {
	err := n.client.controller.Informer().GetIndexer().AddIndexers(map[string]cache.IndexFunc{
		name: func(obj interface{}) ([]string, error) {
			if v, ok := obj.(*Execution); ok {
				return indexer(v)
			}
			return nil, nil
		},
	})

	if err != nil {
		panic(err)
	}
}

func (n *executionClientCache) GetIndexed(name, key string) ([]*Execution, error) {
	var result []*Execution
	objs, err := n.client.controller.Informer().GetIndexer().ByIndex(name, key)
	if err != nil {
		return nil, err
	}
	for _, obj := range objs {
		if v, ok := obj.(*Execution); ok {
			result = append(result, v)
		}
	}

	return result, nil
}

func (n *executionClient2) loadController() {
	if n.controller == nil {
		n.controller = n.iface.Controller()
	}
}

type executionLifecycleDelegate struct {
	create ExecutionChangeHandlerFunc
	update ExecutionChangeHandlerFunc
	remove ExecutionChangeHandlerFunc
}

func (n *executionLifecycleDelegate) HasCreate() bool {
	return n.create != nil
}

func (n *executionLifecycleDelegate) Create(obj *Execution) (runtime.Object, error) {
	if n.create == nil {
		return obj, nil
	}
	return n.create(obj)
}

func (n *executionLifecycleDelegate) HasFinalize() bool {
	return n.remove != nil
}

func (n *executionLifecycleDelegate) Remove(obj *Execution) (runtime.Object, error) {
	if n.remove == nil {
		return obj, nil
	}
	return n.remove(obj)
}

func (n *executionLifecycleDelegate) Updated(obj *Execution) (runtime.Object, error) {
	if n.update == nil {
		return obj, nil
	}
	return n.update(obj)
}
