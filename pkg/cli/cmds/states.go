package cmds

import (
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/terraform/state"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var simpleStateTableHeader = []string{"NAME", "RUNNER NAME", "STATUS"}

func ExecutionCommand() cli.Command {
	return cli.Command{
		Name:    "executions",
		Aliases: []string{"execution"},
		Usage:   "Operations on TF Operator modules",
		Action:  stateList,
		Subcommands: []cli.Command{
			{
				Name:      "ls",
				Usage:     "List Executions",
				ArgsUsage: "None",
				Action:    stateList,
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
						Value: state.DefaultExecutorImage,
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
				Action:    deleteState,
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

func stateList(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	states, err := getStateList(namespace, kubeConfig)
	if err != nil {
		return err
	}

	NewTableWriter(getSimpleStateTableHeader(), stateListToTableStrings(states)).Write()

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

	execution := &v1.State{
		Spec: v1.StateSpec{
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

	return doStateCreate(namespace, kubeConfig, execution)
}

func runExecution(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	state, err := getState(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	// Not really an ideal or safe operation.
	// Need to create something on the execution type to lock
	state.Spec.Version += 1

	_, err = saveState(kubeConfig, namespace, state)
	return err
}

func deleteState(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	executionName := c.Args()[0]

	return doStateDelete(namespace, kubeConfig, executionName)
}

func doStateDelete(namespace, kubeConfig, stateName string) error {
	controllers, err := getControllers(kubeConfig, namespace)

	if err != nil {
		return err
	}
	return controllers.states.Delete(namespace, stateName, &metav1.DeleteOptions{})
}

func doStateCreate(namespace, kubeConfig string, execution *v1.State) error {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return err
	}

	execution.Namespace = namespace

	_, err = controllers.states.Create(execution)
	return err
}

func getState(namespace, kubeConfig, stateName string) (*v1.State, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}
	return controllers.states.Get(namespace, stateName, metav1.GetOptions{})
}

func saveState(kubeConfig, namespace string, state *v1.State) (*v1.State, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}
	return controllers.states.Update(state)
}

func getStateList(namespace, kubeConfig string) (*v1.StateList, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}
	return controllers.states.List(namespace, metav1.ListOptions{})
}

func getSimpleStateTableHeader() []string {
	return simpleStateTableHeader
}

func stateListToTableStrings(states *v1.StateList) [][]string {
	var values [][]string

	for _, execution := range states.Items {
		values = append(values, []string{
			execution.Name,
			execution.Status.ExecutionName,
			execution.Status.Conditions[len(execution.Status.Conditions)-1].Type,
		})
	}

	return values
}
