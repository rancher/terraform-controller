package main

import (
	"github.com/ibuildthecloud/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"github.com/rancher/norman/generator"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const pkg = "github.com/ibuildthecloud/terraform-operator/types"

func main() {
	if err := generator.DefaultGenerate(v1.Schemas, pkg, false, nil); err != nil {
		logrus.Fatal(err)
	}

	if err := generator.ControllersForForeignTypes(pkg, corev1.SchemeGroupVersion,
		[]interface{}{
			corev1.ConfigMap{},
			corev1.Secret{},
		}, nil); err != nil {
		logrus.Fatal(err)
	}
}
