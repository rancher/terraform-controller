package cmds

import (
	"github.com/rancher/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var simpleRunTableHeaders = []string{"RUN NAME", "EXECUTION NAME", "APPROVAL"}

func RunCommand() cli.Command {
	return cli.Command{
		Name:    "runs",
		Aliases: []string{"run"},
		Usage:   "Manage execution runs",
		Action:  runList,
		Subcommands: []cli.Command{
			{
				Name:      "ls",
				Usage:     "List runs",
				ArgsUsage: "None",
				Action:    runList,
			},
			{
				Name:      "approve",
				Usage:     "Approves a run",
				ArgsUsage: "[RUN NAME]",
				Action:    approveRun,
			},
			{
				Name:      "deny",
				Usage:     "Deny approval for a run",
				ArgsUsage: "[RUN NAME]",
				Action:    denyRun,
			},
		},
	}
}

func runList(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	runs, err := getRunList(namespace, kubeConfig)
	if err != nil {
		return err
	}

	NewTableWriter(getSimpleRunTableHeader(), runsListToTableStrings(runs)).Write()

	return nil
}

func approveRun(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	run, err := getRun(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	run.Annotations["approved"] = "yes"

	_, err = saveExecutionRun(kubeConfig, run)
	return err
}

func denyRun(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	run, err := getRun(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	run.Annotations["approved"] = "no"

	_, err = saveExecutionRun(kubeConfig, run)
	return err
}

func getSimpleRunTableHeader() []string {
	return simpleRunTableHeaders
}

func getRunList(namespace, kubeConfig string) (*v1.ExecutionRunList, error) {
	clientSet, err := newV1Client(kubeConfig)
	if err != nil {
		return nil, err
	}

	return clientSet.ExecutionRun.List(namespace, metav1.ListOptions{})
}

func runsListToTableStrings(runs *v1.ExecutionRunList) [][]string {
	var values [][]string

	for _, run := range runs.Items {
		approved := "True"
		if approvalStatus, ok := run.Annotations["approved"]; ok && approvalStatus == "" || approvalStatus == "no" {
			approved = "False"
		}

		values = append(values, []string{
			run.Name,
			run.ObjectMeta.OwnerReferences[0].Name,
			approved,
		})
	}

	return values
}

func saveExecutionRun(kubeConfig string, run *v1.ExecutionRun) (*v1.ExecutionRun, error) {
	clientSet, err := newV1Client(kubeConfig)
	if err != nil {
		return nil, err
	}

	return clientSet.ExecutionRun.Update(run)
}

func getRun(namespace, kubeConfig, name string) (*v1.ExecutionRun, error) {
	clientSet, err := newV1Client(kubeConfig)
	if err != nil {
		return nil, err
	}

	return clientSet.ExecutionRun.Get(namespace, name, metav1.GetOptions{})
}
