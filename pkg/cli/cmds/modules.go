package cmds

import (
	"github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var simpleModuleTableHeader = []string{"NAME", "GIT REPO"}

type InvalidArgs struct{}

func (e InvalidArgs) Error() string {
	return "Invalid args"
}

func ModuleCommand() cli.Command {
	return cli.Command{
		Name:    "modules",
		Aliases: []string{"module"},
		Usage:   "Operations on TF Operator modules",
		Action:  moduleList,
		Subcommands: []cli.Command{
			{
				Name:      "ls",
				Usage:     "List Modules",
				ArgsUsage: "None",
				Action:    moduleList,
			},
			{
				Name:      "create",
				Usage:     "Create new module",
				ArgsUsage: "[NAME] [GIT URL]",
				Action:    createModule,
			},
			{
				Name:      "delete",
				Usage:     "Create new module",
				ArgsUsage: "[NAME]",
				Action:    deleteModule,
			},
		},
	}
}

func moduleList(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	modules, err := getModuleList(namespace, kubeConfig)
	if err != nil {
		return err
	}

	NewTableWriter(getSimpleModuleTableHeader(), moduleListToTableStrings(modules)).Write()

	return nil
}

func createModule(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) != 2 {
		return InvalidArgs{}
	}

	moduleName := c.Args()[0]
	gitUrl := c.Args()[1]

	return doModuleCreate(namespace, kubeConfig, moduleName, gitUrl)
}

func deleteModule(c *cli.Context) error {
	kubeConfig := c.GlobalString("kubeconfig")
	namespace := c.GlobalString("namespace")

	if len(c.Args()) != 1 {
		return InvalidArgs{}
	}

	moduleName := c.Args()[0]

	return doModuleDelete(namespace, kubeConfig, moduleName)
}

func getModuleList(namespace, kubeConfig string) (*v1.ModuleList, error) {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return nil, err
	}
	return controllers.modules.List(namespace, metav1.ListOptions{})

}

func doModuleCreate(namespace, kubeConfig, name, url string) error {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return err
	}

	module := &v1.Module{
		Spec: v1.ModuleSpec{
			ModuleContent: v1.ModuleContent{
				Git: v1.GitLocation{
					URL: url,
				},
			},
		},
	}

	module.Namespace = namespace
	module.Name = name

	_, err = controllers.modules.Create(module)
	return err
}

func doModuleDelete(namespace, kubeConfig, name string) error {
	controllers, err := getControllers(kubeConfig, namespace)
	if err != nil {
		return err
	}

	return controllers.modules.Delete(namespace, name, &metav1.DeleteOptions{})
}

func getSimpleModuleTableHeader() []string {
	return simpleModuleTableHeader
}

func moduleListToTableStrings(modules *v1.ModuleList) [][]string {
	var values [][]string

	for _, module := range modules.Items {
		values = append(values, []string{
			module.Name,
			module.Spec.Git.URL,
		})
	}

	return values
}
