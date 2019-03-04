package main

import (
	"github.com/rancher/norman/generator"
	"github.com/rancher/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
)

const pkg = "github.com/rancher/terraform-operator/types"

func main() {
	if err := generator.DefaultGenerate(v1.Schemas, pkg, false, nil); err != nil {
		logrus.Fatal(err)
	}

	coreObjs := []interface{}{
		corev1.ConfigMap{},
		corev1.Secret{},
		corev1.ServiceAccount{},
	}

	if err := generator.ControllersForForeignTypes(pkg, corev1.SchemeGroupVersion, coreObjs, nil); err != nil {
		logrus.Fatal(err)
	}

	batchObjs := []interface{}{
		batchv1.Job{},
	}

	if err := generator.ControllersForForeignTypes(pkg, batchv1.SchemeGroupVersion, batchObjs, nil); err != nil {
		logrus.Fatal(err)
	}

	rbacObjs := []interface{}{
		rbacV1.ClusterRole{},
		rbacV1.ClusterRoleBinding{},
	}

	if err := generator.ControllersForForeignTypes(pkg, rbacV1.SchemeGroupVersion, nil, rbacObjs); err != nil {
		logrus.Fatal(err)
	}
}
