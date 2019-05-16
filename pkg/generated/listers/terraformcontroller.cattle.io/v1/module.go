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
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ModuleLister helps list Modules.
type ModuleLister interface {
	// List lists all Modules in the indexer.
	List(selector labels.Selector) (ret []*v1.Module, err error)
	// Modules returns an object that can list and get Modules.
	Modules(namespace string) ModuleNamespaceLister
	ModuleListerExpansion
}

// moduleLister implements the ModuleLister interface.
type moduleLister struct {
	indexer cache.Indexer
}

// NewModuleLister returns a new ModuleLister.
func NewModuleLister(indexer cache.Indexer) ModuleLister {
	return &moduleLister{indexer: indexer}
}

// List lists all Modules in the indexer.
func (s *moduleLister) List(selector labels.Selector) (ret []*v1.Module, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Module))
	})
	return ret, err
}

// Modules returns an object that can list and get Modules.
func (s *moduleLister) Modules(namespace string) ModuleNamespaceLister {
	return moduleNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ModuleNamespaceLister helps list and get Modules.
type ModuleNamespaceLister interface {
	// List lists all Modules in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.Module, err error)
	// Get retrieves the Module from the indexer for a given namespace and name.
	Get(name string) (*v1.Module, error)
	ModuleNamespaceListerExpansion
}

// moduleNamespaceLister implements the ModuleNamespaceLister
// interface.
type moduleNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Modules in the indexer for a given namespace.
func (s moduleNamespaceLister) List(selector labels.Selector) (ret []*v1.Module, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Module))
	})
	return ret, err
}

// Get retrieves the Module from the indexer for a given namespace and name.
func (s moduleNamespaceLister) Get(name string) (*v1.Module, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("module"), name)
	}
	return obj.(*v1.Module), nil
}
