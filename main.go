//go:generate go run pkg/codegen/cleanup/main.go
//go:generate /bin/rm -rf pkg/generated
//go:generate go run pkg/codegen/main.go

package main

import (
	"context"
	"os"

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
		cli.BoolFlag{
			Name:   "enable-api",
			EnvVar: "API",
			Usage:  "Enables the remote backend api functionality",
		},
		cli.BoolTFlag{
			Name:   "enable-controller",
			EnvVar: "CONTROLLER",
			Usage:  "Enables the kubernetes controller functionality",
		},
		cli.StringFlag{
			Name:   "api-address",
			EnvVar: "API_ADDRESS",
			Value:  "0.0.0.0:8080",
			Usage:  "Address to run the REST api",
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

	if c.BoolT("enable-controller") {
		go startController(ctx, c)
	}
	if c.Bool("enable-api") {
		go startAPI(ctx, c)
	}

	<-ctx.Done()
}

func startController(ctx context.Context, c *cli.Context) {
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

	if err := controller.Start(ctx, kubeconfig, ns, masterurl, threadiness); err != nil {
		logrus.Fatalf("failed to start controller: %s", err.Error())
	}
}

func startAPI(ctx context.Context, c *cli.Context) {
	if err := api.Start(ctx, c.String("api-address")); err != nil {
		logrus.Fatalf("failed to start api: %s", err.Error())
	}
}
