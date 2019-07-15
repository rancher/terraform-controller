package e2e

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
)

func TestCrds(t *testing.T) {
	assert := assert.New(t)
	cs, err := clientset.NewForConfig(e.cfg)
	assert.Nil(err)
	for _, c := range e.crds {
		crdName := fmt.Sprintf("%s.%s", lowerPlural(c.GVK.Kind), c.GVK.Group)
		crd, err := cs.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})
		assert.Nil(err)
		assert.Equal(crdName, crd.Name)
	}
}

func lowerPlural(s string) string {
	lcp := namer.NewAllLowercasePluralNamer(map[string]string{})
	return lcp.Name(&types.Type{
		Name: types.Name{
			Name: s,
		},
	})
}

func TestGetNs(t *testing.T) {
	assert := assert.New(t)
	ns := e.getNs()
	assert.Equal(reflect.TypeOf(ns), reflect.TypeOf(&corev1.Namespace{}))
	assert.Equal(ns.ObjectMeta.Name, e.namespace)
}

func TestGetSa(t *testing.T) {
	assert := assert.New(t)
	sa := e.getSa()
	assert.Equal(reflect.TypeOf(sa), reflect.TypeOf(&corev1.ServiceAccount{}))
	assert.Equal(sa.ObjectMeta.Name, e.namespace)
	assert.Equal(sa.ObjectMeta.Namespace, e.namespace)
}

func TestGetCrb(t *testing.T) {
	assert := assert.New(t)
	crb := e.getCrb()
	assert.Equal(reflect.TypeOf(crb), reflect.TypeOf(&rbacv1.ClusterRoleBinding{}))
	assert.Equal(crb.ObjectMeta.Name, e.namespace)
	assert.Equal(crb.Subjects[0].Kind, "ServiceAccount")
	assert.Equal(crb.Subjects[0].Name, e.namespace)
	assert.Equal(crb.Subjects[0].Namespace, e.namespace)
}
