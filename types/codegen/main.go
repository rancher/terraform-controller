package main

import (
	"github.com/rancher/kerraform/types/apis/kerraform.cattle.io/v1"
	"github.com/rancher/norman/generator"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const pkg = "github.com/rancher/kerraform/types"

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
