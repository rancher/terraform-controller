package cmds

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/docker/go-units"
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
				Name:      "prune",
				Aliases:   []string{"pr"},
				Usage:     "Takes a state and will delete all executions aged > <days> , pass --days to change how many days to leave",
				ArgsUsage: "[STATE NAME]",
				Action:    executionPrune,
				Flags: []cli.Flag{
					cli.IntFlag{
						Name:  "days",
						Usage: "How many days of executions you want to keep around, --days 2 will leave the last 48hrs of executions",
						Value: 7,
					},
				},
			},
			{
				Name:      "logs",
				Aliases:   []string{"l"},
				Usage:     "List executions",
				ArgsUsage: "[EXECUTION NAME]",
				Action:    logs,
			},
			{
				Name:      "delete",
				Aliases:   []string{"del"},
				Usage:     "Delete an execution",
				ArgsUsage: "[EXECUTION NAME]",
				Action:    delete,
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

func executionPrune(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	days := c.Int("days")

	if days > 0 {
		days = days * -1
	}

	now := time.Now()
	from := now.AddDate(0, 0, days)

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
		if exec.CreationTimestamp.After(from) {
			continue
		}
		logrus.Info("deleting " + exec.Name)
		err := controllers.executions.Delete(namespace, exec.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func delete(c *cli.Context) error {
	namespace := c.GlobalString("namespace")
	kubeConfig := c.GlobalString("kubeconfig")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args().First()

	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return err
	}

	_, err = controllers.executions.Get(namespace, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	err = controllers.executions.Delete(namespace, name, &metav1.DeleteOptions{})

	if err != nil {
		return err
	}

	fmt.Printf("Deleted %s\n", name)
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

		age := units.HumanDuration(time.Now().UTC().Sub(execution.ObjectMeta.CreationTimestamp.Time))
		values = append(values, []string{
			execution.Name,
			execution.Labels["state"],
			approved,
			age,
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
