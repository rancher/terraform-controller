//go:generate go run pkg/codegen/cleanup/main.go
//go:generate /bin/rm -rf pkg/generated
//go:generate go run pkg/codegen/main.go

package main

import (
	"context"
	"os"

	"github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io"
	"github.com/rancher/terraform-controller/pkg/terraform"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/batch"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/core"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/rbac"
	"github.com/rancher/wrangler/pkg/resolvehome"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	VERSION = "v0.0.0-dev"
)

func main() {
	app := cli.NewApp()
	app.Name = "terraform-controller"
	app.Version = VERSION
	app.Flags = []cli.Flag{

		cli.IntFlag{
			Name:   "threads",
			EnvVar: "THREADS",
			Value:  2,
		},
		cli.BoolFlag{
			Name:   "debug",
			EnvVar: "DEBUG",
		},
		cli.StringFlag{
			Name:   "kubeconfig",
			EnvVar: "KUBECONFIG",
			Value:  "${HOME}/.kube/config",
		},
		cli.StringFlag{
			Name:   "namespace",
			EnvVar: "NAMESPACE",
			Value:  "default",
		},
		cli.StringFlag{
			Name:   "masterurl",
			EnvVar: "MASTERURL",
			Value:  "",
		},
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) {

	if c.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetReportCaller(true)
	}

	logrus.Info("Starting Terraform Controller")
	kubeconfig, err := resolvehome.Resolve(c.String("kubeconfig"))

	if err != nil {
		logrus.Info("Resolving home dir failed.")
	}

	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		kubeconfig = ""
	}

	threadiness := c.Int("threads")
	masterurl := c.String("masterurl")
	ns := c.String("namespace")

	logrus.Printf("Booting Terraform Controller, namespace: %s", ns)

	ctx := signals.SetupSignalHandler(context.Background())

	cfg, err := clientcmd.BuildConfigFromFlags(masterurl, kubeconfig)
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	tfFactory, err := terraformcontroller.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		logrus.Fatalf("Error building terraform controllers: %s", err.Error())
	}

	coreFactory, err := core.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		logrus.Fatalf("Error building core controllers: %s", err.Error())
	}

	rbacFactory, err := rbac.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		logrus.Fatalf("Error building rbac controllers: %s", err.Error())
	}

	batchFactory, err := batch.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		logrus.Fatalf("Error building rbac controllers: %s", err.Error())
	}

	terraform.Register(ctx,
		tfFactory.Terraformcontroller().V1().Module(),
		tfFactory.Terraformcontroller().V1().State(),
		tfFactory.Terraformcontroller().V1().Execution(),
		rbacFactory.Rbac().V1().ClusterRole(),
		rbacFactory.Rbac().V1().ClusterRoleBinding(),
		coreFactory.Core().V1().Secret(),
		coreFactory.Core().V1().ConfigMap(),
		coreFactory.Core().V1().ServiceAccount(),
		batchFactory.Batch().V1().Job(),
	)

	if err := start.All(ctx, threadiness, tfFactory, coreFactory, rbacFactory, batchFactory); err != nil {
		logrus.Fatalf("Error starting: %s", err.Error())
	}

	<-ctx.Done()
}
