package main

import (
	"os"

	"github.com/rancher/terraform-operator/pkg/cli/cmds"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	VERSION = "v0.0.0-dev"
)

func main() {
	app := cli.NewApp()
	app.Name = "tffy"
	app.Version = VERSION
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "kubeconfig",
			EnvVar: "KUBECONFIG",
			Value:  "${HOME}/.kube/config",
		},
		cli.StringFlag{
			Name:   "namespace,n",
			EnvVar: "NAMESPACE",
			Value:  "default",
		},
	}

	app.Commands = []cli.Command{
		cmds.ModuleCommand(),
		cmds.ExecutionCommand(),
		cmds.RunCommand(),
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {

	return nil
}
