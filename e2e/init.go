package e2e

import (
	"time"

	"github.com/rancher/wrangler/pkg/crd"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (e *E2E) initialize() error {
	var err error

	crdFactory, err := crd.NewFactoryFromClient(e.cfg)
	if err != nil {
		logrus.Fatalf("Error building crd: %s", err.Error())
	}

	_, err = crdFactory.CreateCRDs(e.ctx, e.crds...)
	if err != nil {
		return err
	}

	_, err = e.cs.CoreV1().Namespaces().Create(e.ctx, e.getNs(), v13.CreateOptions{})
	if err != nil {
		return err
	}
	_, err = e.cs.RbacV1().ClusterRoleBindings().Create(e.ctx, e.getCrb(), v13.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = e.cs.CoreV1().ServiceAccounts(e.namespace).Create(e.ctx, e.getSa(), v13.CreateOptions{})
	if err != nil {
		return err
	}

	err = wait.Poll(time.Second, 15*time.Second, func() (bool, error) {
		_, err := e.cs.CoreV1().ServiceAccounts(e.namespace).Get(e.ctx, e.namespace, v13.GetOptions{})
		if err == nil {
			return true, nil
		}

		if errors.IsNotFound(err) {
			return false, nil
		}

		logrus.Printf("Waiting for SA to be ready: %+v\n", err)
		return false, err
	})

	if err != nil {
		return err
	}

	return nil
}

func (e *E2E) getNs() *v12.Namespace {
	return &v12.Namespace{
		ObjectMeta: v13.ObjectMeta{
			Name: e.namespace,
		},
	}
}

func (e *E2E) getSa() *v12.ServiceAccount {
	return &v12.ServiceAccount{
		ObjectMeta: v13.ObjectMeta{
			Name:      e.namespace,
			Namespace: e.namespace,
			Labels: map[string]string{
				"apps.kubernetes.io/component": "controller",
				"apps.kubernetes.io/name":      e.namespace,
			},
		},
	}
}

func (e *E2E) getCrb() *v1.ClusterRoleBinding {
	return &v1.ClusterRoleBinding{
		ObjectMeta: v13.ObjectMeta{
			Name: e.namespace,
			Labels: map[string]string{
				"apps.kubernetes.io/component": "controller",
				"apps.kubernetes.io/name":      e.namespace,
			},
		},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      e.namespace,
				Namespace: e.namespace,
			},
		},
	}
}
