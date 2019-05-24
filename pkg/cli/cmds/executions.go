package cmds

import (
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var simpleRunTableHeaders = []string{"RUN NAME", "EXECUTION NAME", "APPROVAL"}

func ExecutionCommand() cli.Command {
	return cli.Command{
		Name:    "executions",
		Aliases: []string{"execution"},
		Usage:   "Manage executions",
		Action:  executionList,
		Subcommands: []cli.Command{
			{
				Name:      "ls",
				Usage:     "List executions",
				ArgsUsage: "None",
				Action:    executionList,
			},
			{
				Name:      "approve",
				Usage:     "Approves an execution",
				ArgsUsage: "[RUN NAME]",
				Action:    approveExecution,
			},
			{
				Name:      "deny",
				Usage:     "Deny approval for an execution",
				ArgsUsage: "[RUN NAME]",
				Action:    denyExecution,
			},
		},
	}
}

func executionList(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	runs, err := getRunList(namespace, kubeConfig)
	if err != nil {
		return err
	}

	NewTableWriter(getSimpleRunTableHeader(), runsListToTableStrings(runs)).Write()

	return nil
}

func approveExecution(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	run, err := getExecution(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	run.Annotations["approved"] = "yes"

	_, err = saveExecution(kubeConfig, namespace, run)
	return err
}

func denyExecution(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	run, err := getExecution(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	run.Annotations["approved"] = "no"

	_, err = saveExecution(kubeConfig, namespace, run)
	return err
}

func getSimpleRunTableHeader() []string {
	return simpleRunTableHeaders
}

func getRunList(namespace, kubeConfig string) (*v1.ExecutionList, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	return controllers.executions.List(namespace, metav1.ListOptions{})
}

func runsListToTableStrings(runs *v1.ExecutionList) [][]string {
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

func saveExecution(kubeConfig, namespace string, run *v1.Execution) (*v1.Execution, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	return controllers.executions.Update(run)
}

func getExecution(namespace, kubeConfig, name string) (*v1.Execution, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	return controllers.executions.Get(namespace, name, metav1.GetOptions{})
}
