package main

import (
	"github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
)

func main() {
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/terraform-controller/pkg/generated",
		Boilerplate:   "hack/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"terraformcontroller.cattle.io": {
				Types: []interface{}{
					v1.Module{},
					v1.Execution{},
					v1.ExecutionRun{},
				},
				GenerateTypes: true,
			},
			"core": {
				Types: []interface{}{
					corev1.ConfigMap{},
					corev1.Secret{},
					corev1.ServiceAccount{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
			"batch": {
				Types: []interface{}{
					batchv1.Job{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
			"rbac": {
				Types: []interface{}{
					rbacV1.ClusterRole{},
					rbacV1.ClusterRoleBinding{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
		},
	})
}
