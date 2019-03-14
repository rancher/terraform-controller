package cmds

import (
	"github.com/rancher/terraform-operator/pkg/controllers/execution"
	"github.com/rancher/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var simpleExecutionTableHeader = []string{"NAME", "RUNNER NAME", "STATUS"}

func ExecutionCommand() cli.Command {
	return cli.Command{
		Name:    "executions",
		Aliases: []string{"module"},
		Usage:   "Operations on TF Operator modules",
		Action:  executionList,
		Subcommands: []cli.Command{
			{
				Name:      "ls",
				Usage:     "List Executions",
				ArgsUsage: "None",
				Action:    executionList,
			},
			{
				Name:      "create",
				Usage:     "Create new executions pointing to a module",
				ArgsUsage: "[EXECUTION NAME] [MODULE NAME]",
				Action:    createExecution,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "image",
						Usage: "Set the image for the execution environment [registry]:org/repo:tag",
						Value: execution.DefaultExecutorImage,
					},
					cli.BoolFlag{
						Name:  "destroy-on-delete",
						Usage: "If this execution is deleted a TF destroy is also run",
					},
					cli.BoolFlag{
						Name:  "autoconfirm",
						Usage: "Autoapply TF updates",
					},
					cli.StringSliceFlag{
						Name:  "secret",
						Usage: "Name of Kubernetes secret to use during execution run(Must be in same namespace and pre-created)",
					},
					cli.StringSliceFlag{
						Name:  "configmap",
						Usage: "Name of Kubernetes configmap to use during execution run(Must be in same namespace and pre-created)",
					},
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete execution",
				ArgsUsage: "[EXECUTION NAME]",
				Action:    deleteExecution,
			},
			{
				Name:      "run",
				Usage:     "Run the execution",
				Action:    runExecution,
				ArgsUsage: "[EXECUTION NAME]",
			},
		},
	}
}

func executionList(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	executions, err := getExecutionList(namespace, kubeConfig)
	if err != nil {
		return err
	}

	NewTableWriter(getSimpleExecutionTableHeader(), executionListToTableStrings(executions)).Write()

	return nil
}

func createExecution(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) != 2 {
		return InvalidArgs{}
	}

	executionName := c.Args()[0]
	moduleName := c.Args()[1]

	execution := &v1.Execution{
		Spec: v1.ExecutionSpec{
			ModuleName:      moduleName,
			Image:           c.String("image"),
			DestroyOnDelete: c.Bool("destroy-on-delete"),
			AutoConfirm:     c.Bool("autoconfirm"),
			Variables: v1.Variables{
				SecretNames:   c.StringSlice("secret"),
				EnvConfigName: c.StringSlice("configmap"),
			},
		},
	}

	execution.Name = executionName

	return doExecutionCreate(namespace, kubeConfig, execution)
}

func runExecution(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	execution, err := getExecution(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	// Not really an ideal or safe operation.
	// Need to create something on the execution type to lock
	execution.Spec.Version += 1

	_, err = saveExecution(kubeConfig, execution)
	return err
}

func deleteExecution(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	executionName := c.Args()[0]

	return doExecutionDelete(namespace, kubeConfig, executionName)
}

func doExecutionDelete(namespace, kubeConfig, executionName string) error {
	clientSet, err := newV1Client(kubeConfig)
	if err != nil {
		return err
	}
	return clientSet.Execution.Delete(namespace, executionName, &metav1.DeleteOptions{})
}

func doExecutionCreate(namespace, kubeConfig string, execution *v1.Execution) error {
	clientSet, err := newV1Client(kubeConfig)
	if err != nil {
		return err
	}

	execution.Namespace = namespace

	_, err = clientSet.Execution.Create(execution)
	return err
}

func getExecution(namespace, kubeConfig, executionName string) (*v1.Execution, error) {
	clientSet, err := newV1Client(kubeConfig)
	if err != nil {
		return nil, err
	}
	return clientSet.Execution.Get(namespace, executionName, metav1.GetOptions{})
}

func saveExecution(kubeConfig string, execution *v1.Execution) (*v1.Execution, error) {
	clientSet, err := newV1Client(kubeConfig)
	if err != nil {
		return nil, err
	}
	return clientSet.Execution.Update(execution)
}

func getExecutionList(namespace, kubeConfig string) (*v1.ExecutionList, error) {
	clientSet, err := newV1Client(kubeConfig)
	if err != nil {
		return nil, err
	}
	return clientSet.Execution.List(namespace, metav1.ListOptions{})
}

func getSimpleExecutionTableHeader() []string {
	return simpleExecutionTableHeader
}

func executionListToTableStrings(executions *v1.ExecutionList) [][]string {
	var values [][]string

	for _, execution := range executions.Items {
		values = append(values, []string{
			execution.Name,
			execution.Status.ExecutionRunName,
		})
	}

	return values
}
