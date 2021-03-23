//go:generate go run pkg/codegen/cleanup/main.go
//go:generate /bin/rm -rf pkg/generated
//go:generate go run pkg/codegen/main.go

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rancher/wrangler/pkg/start"

	"github.com/rancher/terraform-controller/pkg/types"

	"github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io"
	"github.com/rancher/wrangler/pkg/generated/controllers/batch"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/generated/controllers/rbac"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/rancher/terraform-controller/pkg/api"
	"github.com/rancher/terraform-controller/pkg/controller"
	"github.com/rancher/wrangler/pkg/resolvehome"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
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
			Usage:  "Set the threadiness of the kubernetes controller",
		},
		cli.BoolFlag{
			Name:   "debug",
			EnvVar: "DEBUG",
			Usage:  "Enables debug output",
		},
		cli.StringFlag{
			Name:   "api-address",
			EnvVar: "API_ADDRESS",
			Value:  "0.0.0.0:8080",
			Usage:  "Address to run the REST api",
		},
		cli.StringFlag{
			Name:   "api-cert-file",
			EnvVar: "API_CERT_FILE",
			Value:  "",
			Usage:  "A pem cert file for TLS for the REST API, leave blank for no TLS",
		},
		cli.StringFlag{
			Name:   "api-key-file",
			EnvVar: "API_KEY_FILE",
			Value:  "",
			Usage:  "A pem key file for TLS for the REST API, leave blank for no TLS",
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
			Usage:  "The namespace to run the kubernetes controller in",
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
	ctx := signals.SetupSignalHandler(context.Background())

	controllers, factories, err := loadControllers(c)
	if err != nil {
		log.Fatalf("error building controllers: %s", err)
	}

	startController(ctx, c, controllers)
	startFactories(ctx, c, factories)
	go startAPI(ctx, c, controllers) // http server blocks so run in a routine

	<-ctx.Done()
}

func startController(ctx context.Context, c *cli.Context, controllers *types.Controllers) {
	if err := controller.Start(ctx,
		controllers,
		c.String("namespace")); err != nil {
		logrus.Fatalf("failed to start controller: %s", err.Error())
	}
}

func startFactories(ctx context.Context, c *cli.Context, factories *types.Factories) {
	if err := start.All(ctx, c.Int("threads"), factories.Tf, factories.Core, factories.Rbac, factories.Batch); err != nil {
		log.Fatalf("failed to start all factories: %s", err)
	}
}

func startAPI(ctx context.Context, c *cli.Context, controllers *types.Controllers) {
	if err := api.Start(ctx,
		controllers,
		c.String("api-address"),
		c.String("api-cert-file"),
		c.String("api-key-file")); err != nil {
		logrus.Fatalf("failed to start api: %s", err.Error())
	}
}

func loadControllers(c *cli.Context) (*types.Controllers, *types.Factories, error) {
	kubeconfig, err := resolvehome.Resolve(c.String("kubeconfig"))
	if err != nil {
		logrus.Info("Resolving home dir failed.")
	}

	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		kubeconfig = ""
	}

	ns := c.String("namespace")
	masterurl := c.String("masterurl")
	cfg, err := clientcmd.BuildConfigFromFlags(masterurl, kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("error building kubeconfig: %s", err.Error())
	}

	tfFactory, err := terraformcontroller.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("error building controller controllers: %s", err.Error())
	}

	coreFactory, err := core.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("error building core controllers: %s", err.Error())
	}

	rbacFactory, err := rbac.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("error building rbac controllers: %s", err.Error())
	}

	batchFactory, err := batch.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("error building rbac controllers: %s", err.Error())
	}

	return &types.Controllers{
			Module:             tfFactory.Terraformcontroller().V1().Module(),
			State:              tfFactory.Terraformcontroller().V1().State(),
			Execution:          tfFactory.Terraformcontroller().V1().Execution(),
			ClusterRole:        rbacFactory.Rbac().V1().ClusterRole(),
			ClusterRoleBinding: rbacFactory.Rbac().V1().ClusterRoleBinding(),
			Secret:             coreFactory.Core().V1().Secret(),
			ConfigMap:          coreFactory.Core().V1().ConfigMap(),
			ServiceAccount:     coreFactory.Core().V1().ServiceAccount(),
			Job:                batchFactory.Batch().V1().Job(),
		}, &types.Factories{
			Tf:    tfFactory,
			Core:  coreFactory,
			Rbac:  rbacFactory,
			Batch: batchFactory,
		}, nil
}
