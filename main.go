//go:generate go run types/codegen/cleanup/main.go
//go:generate go run types/codegen/main.go

package main

import (
	"context"
	"os"

	"github.com/rancher/norman"
	"github.com/rancher/norman/pkg/resolvehome"
	"github.com/rancher/norman/signal"
	"github.com/rancher/terraform-operator/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	VERSION = "v0.0.0-dev"
)

func main() {
	app := cli.NewApp()
	app.Name = "terraform-operator"
	app.Version = VERSION
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name: "external",
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
			Name:   "log-level",
			EnvVar: "LOG_LEVEL",
			Value:  "info",
		},
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	logrus.Info("Starting controller")
	k8sMode := "auto"
	kubeConfig := ""

	level, err := logrus.ParseLevel(c.String("log-level"))
	if err != nil {
		return err
	}
	logrus.SetLevel(level)

	ctx := signal.SigTermCancelContext(context.Background())

	if c.Bool("external") {
		kubeConfig, err = resolvehome.Resolve(c.String("kubeconfig"))
		if err != nil {
			return err
		}
		k8sMode = "external"
	}

	ns := c.String("namespace")
	ctx, _, err = server.Config(ns).Build(ctx, &norman.Options{
		K8sMode:    k8sMode,
		KubeConfig: kubeConfig,
	})

	if err != nil {
		return err
	}
	<-ctx.Done()

	return nil
}
