package cmds

import (
	"fmt"

	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/terraform/state"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var simpleStateTableHeader = []string{"NAME", "RUNNER NAME", "STATUS"}

func StateCommand() cli.Command {
	return cli.Command{
		Name:    "states",
		Aliases: []string{"state", "s"},
		Usage:   "Manage states",
		Action:  stateList,
		Subcommands: []cli.Command{
			{
				Name:      "list",
				Usage:     "List States test",
				ArgsUsage: "None",
				Action:    stateList,
				Aliases:   []string{"ls"},
			},
			{
				Name:      "show",
				Usage:     "Show state, gather all variables/secrets/modules and display them.",
				ArgsUsage: "[STATE NAME]",
				Action:    stateShow,
			},
			{
				Name:      "unlock",
				Usage:     "Clear State File Lock",
				ArgsUsage: "[STATE]",
				Action:    stateUnlock,
				Aliases:   []string{"su"},
			},
			{
				Name:      "create",
				Usage:     "Create new state pointing to a module",
				ArgsUsage: "[EXECUTION NAME] [MODULE NAME]",
				Action:    createState,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "image",
						Usage: "Set the image for the execution environment [registry]:org/repo:tag",
						Value: state.DefaultExecutorImage,
					},
					cli.BoolFlag{
						Name:  "destroy-on-delete",
						Usage: "If this state is deleted a TF destroy is also run",
					},
					cli.BoolFlag{
						Name:  "autoconfirm",
						Usage: "Autoapply TF updates",
					},
					cli.StringSliceFlag{
						Name:  "secret",
						Usage: "Name of Kubernetes secret to use during execution (Must be in same namespace and pre-created)",
					},
					cli.StringSliceFlag{
						Name:  "configmap",
						Usage: "Name of Kubernetes configmap to use during execution (Must be in same namespace and pre-created)",
					},
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete state.",
				ArgsUsage: "[STATE NAME]",
				Aliases:   []string{"del", "d"},
				Action:    deleteState,
			},
			{
				Name:      "run",
				Usage:     "Run the state, will refresh variables and create an execution.",
				Action:    runState,
				ArgsUsage: "[STATE NAME]",
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

func stateShow(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) < 1 {
		logrus.Fatal("No state name passed.")
	}

	name := c.Args()[0]

	state, err := getState(namespace, kubeConfig, name)
	if err != nil {
		return err
	}

	fmt.Printf("State: %s\n", name)
	fmt.Printf("Auto Confirm: %t\n", state.Spec.AutoConfirm)
	fmt.Printf("Destroy on Deleted: %t\n\n", state.Spec.AutoConfirm)

	if len(state.Spec.Variables.EnvConfigName) > 0 {
		fmt.Print("\n")
		for _, value := range state.Spec.Variables.EnvConfigName {
			config, err := getConfig(namespace, kubeConfig, value)
			if err != nil {
				if k8sError.IsNotFound(err) {
					fmt.Printf("missing info: secret %s\n", value)
					continue
				}
			}

			for key, value := range config.Data {
				fmt.Printf("%s: %s\n", key, value)
			}
		}
	}

	if len(state.Spec.Variables.SecretNames) > 0 {
		fmt.Print("\n")
		for _, secretName := range state.Spec.Variables.SecretNames {
			secret, err := getSecret(namespace, kubeConfig, secretName)
			if err != nil {
				if k8sError.IsNotFound(err) {
					fmt.Printf("missing info: secret %s\n", secretName)
					continue
				}
			}

			for key, value := range secret.Data {
				fmt.Printf("%s: %s\n", key, value)
			}
		}
	}

	return nil
}

func stateUnlock(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return err
	}

	args := c.Args()
	if len(args) != 1 {
		return cli.NewExitError("No state name passed", 1)
	}
	name := c.Args().First()

	selector := fmt.Sprintf("%s=true,%s=%s", terraState, terraKey, name)
	secrets, err := controllers.secrets.List(namespace, metav1.ListOptions{
		LabelSelector: selector,
	})

	if err != nil {
		return err
	}

	for _, secret := range secrets.Items {
		if secret.Data["lockInfo"] != nil {
			copyObj := secret.DeepCopy()
			copyObj.Data["lockInfo"] = nil
			_, err := controllers.secrets.Update(copyObj)
			if err != nil {
				return err
			}
			logrus.Infof("Unlocking %s", name)
		}
	}

	return nil
}

func createState(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) != 2 {
		return cli.NewExitError("Expects two params.", 1)
	}

	stateName := c.Args()[0]
	moduleName := c.Args()[1]

	state := &v1.State{
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

	state.Name = stateName

	return doStateCreate(namespace, kubeConfig, state)
}

func runState(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	name := c.Args()[0]

	state, err := getState(namespace, kubeConfig, name)
	if err != nil {
		logrus.Debug(err)
		return err
	}

	copyObj := state.DeepCopy()
	copyObj.Spec.Version += 1
	copyObj.Status.LastRunHash = ""
	v1.StateConditionJobDeployed.False(copyObj)

	_, err = saveState(kubeConfig, namespace, copyObj)
	return err
}

func deleteState(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")
	controllers, err := getControllers(kubeConfig, namespace)

	if err != nil {
		return err
	}

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	stateName := c.Args()[0]

	state, err := getState(namespace, kubeConfig, stateName)
	if err != nil {
		return err
	}

	if state.DeletionTimestamp != nil {
		state.Status.ExecutionName = ""
		state.Spec.Version += 1
		state.Status.LastRunHash = ""
		v1.StateConditionJobDeployed.False(state)
		_, err = controllers.states.Update(state)
		if err != nil {
			return err
		}
	}

	return controllers.states.Delete(namespace, stateName, &metav1.DeleteOptions{})
}

func doStateCreate(namespace, kubeConfig string, state *v1.State) error {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return err
	}

	state.Namespace = namespace

	_, err = controllers.states.Create(state)
	return err
}

func getState(namespace, kubeConfig, stateName string) (*v1.State, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	return controllers.states.Get(namespace, stateName, metav1.GetOptions{})
}

func getConfig(namespace, kubeConfig, configName string) (*corev1.ConfigMap, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	return controllers.configMaps.Get(namespace, configName, metav1.GetOptions{})
}

func getSecret(namespace, kubeConfig, secret string) (*corev1.Secret, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	return controllers.secrets.Get(namespace, secret, metav1.GetOptions{})
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

	for _, state := range states.Items {
		status := ""
		if 0 < len(state.Status.Conditions) {
			status = string(state.Status.Conditions[0].Status)
		}
		values = append(values, []string{
			state.Name,
			state.Status.ExecutionName,
			status,
		})
	}

	return values
}
