package main

import (
	"github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
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
		},
	})
}
