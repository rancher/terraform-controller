package main

import (
	"fmt"
	"os"

	"github.com/rancher/terraform-controller/pkg/cli/cmds"
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
	app.Action = cli.ShowCommandHelp
	app.HideHelp = false
	app.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Fprintf(c.App.Writer, "Thar be no %q here.\n", command)
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "kubeconfig, k",
			EnvVar: "KUBECONFIG",
			Value:  "${HOME}/.kube/config",
		},
		cli.StringFlag{
			Name:   "namespace, n",
			EnvVar: "NAMESPACE",
			Value:  "default",
		},
	}

	app.Commands = []cli.Command{
		cmds.ModuleCommand(),
		cmds.StateCommand(),
		cmds.ExecutionCommand(),
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	err := cli.ShowAppHelp(c)
	if err != nil {
		return err
	}

	return nil
}
