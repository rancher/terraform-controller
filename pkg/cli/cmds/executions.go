package cmds

import (
	"encoding/base64"
	"fmt"

	"github.com/rancher/terraform-controller/pkg/age"
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/gz"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var simpleRunTableHeaders = []string{"EXECUTION NAME", "STATE NAME", "APPROVAL", "AGE"}

func ExecutionCommand() cli.Command {
	return cli.Command{
		Name:    "executions",
		Aliases: []string{"execution", "exec", "e"},
		Usage:   "Manage executions",
		Action:  executionList,
		Subcommands: []cli.Command{
			{
				Name:      "list",
				Aliases:   []string{"ls"},
				Usage:     "List executions",
				ArgsUsage: "None",
				Action:    executionList,
			},
			{
				Name:      "delete",
				Aliases:   []string{"del"},
				Usage:     "Delete executions for a specific state.",
				ArgsUsage: "[EXECUTION NAME]",
				Action:    executionDelete,
			},
			{
				Name:      "logs",
				Aliases:   []string{"l"},
				Usage:     "List executions",
				ArgsUsage: "[EXECUTION NAME]",
				Action:    logs,
			},
			{
				Name:      "approve",
				Aliases:   []string{"a"},
				Usage:     "Approves an execution",
				ArgsUsage: "[EXECUTION NAME]",
				Action:    approveExecution,
			},
			{
				Name:      "deny",
				Aliases:   []string{"d"},
				Usage:     "Deny approval for an execution",
				ArgsUsage: "[EXECUTION NAME]",
				Action:    denyExecution,
			},
		},
	}
}

func logs(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	execution, err := getExecution(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	compressedLog, err := base64.StdEncoding.DecodeString(execution.Status.JobLogs)
	if err != nil {
		return err
	}

	log, err := gz.Uncompress(compressedLog)
	if err != nil {
		return err
	}

	fmt.Print(string(log))

	return nil
}

func executionDelete(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return err
	}

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	execs, err := controllers.executions.List(namespace, metav1.ListOptions{
		LabelSelector: "state=" + name,
	})

	if err != nil {
		return err
	}

	for _, exec := range execs.Items {
		err := controllers.executions.Delete(namespace, exec.Name, &metav1.DeleteOptions{})
		logrus.Info("deleting " + exec.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func executionList(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	listoptions := metav1.ListOptions{}
	if len(c.Args()) == 1 {
		listoptions = metav1.ListOptions{
			LabelSelector: "state=" + c.Args().First(),
		}
	}

	executions, err := getExecutionList(namespace, kubeConfig, listoptions)
	if err != nil {
		return err
	}

	NewTableWriter(getSimpleRunTableHeader(), runsListToTableStrings(executions)).Write()

	return nil
}

func approveExecution(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	execution, err := getExecution(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	execution.Annotations["approved"] = "yes"

	_, err = saveExecution(kubeConfig, namespace, execution)
	return err
}

func denyExecution(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	execution, err := getExecution(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	execution.Annotations["approved"] = "no"

	_, err = saveExecution(kubeConfig, namespace, execution)
	return err
}

func getSimpleRunTableHeader() []string {
	return simpleRunTableHeaders
}

func getExecutionList(namespace, kubeConfig string, listoptions metav1.ListOptions) (*v1.ExecutionList, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	return controllers.executions.List(namespace, listoptions)
}

func runsListToTableStrings(executions *v1.ExecutionList) [][]string {
	var values [][]string

	for _, execution := range executions.Items {
		approved := "True"
		if approvalStatus, ok := execution.Annotations["approved"]; ok && approvalStatus == "" || approvalStatus == "no" {
			approved = "False"
		}

		values = append(values, []string{
			execution.Name,
			execution.Labels["state"],
			approved,
			age.Age(execution.ObjectMeta.CreationTimestamp.Time),
		})
	}

	return values
}

func saveExecution(kubeConfig, namespace string, execution *v1.Execution) (*v1.Execution, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	return controllers.executions.Update(execution)
}

func getExecution(namespace, kubeConfig, name string) (*v1.Execution, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	return controllers.executions.Get(namespace, name, metav1.GetOptions{})
}
