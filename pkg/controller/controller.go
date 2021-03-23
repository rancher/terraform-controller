package controller

import (
	"context"
	"fmt"

	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/controller/execution"
	"github.com/rancher/terraform-controller/pkg/controller/module"
	"github.com/rancher/terraform-controller/pkg/controller/state"
	"github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io"
	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/batch"
	batchv1 "github.com/rancher/wrangler/pkg/generated/controllers/batch/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	corev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/rbac"
	rbacv1 "github.com/rancher/wrangler/pkg/generated/controllers/rbac/v1"
	"github.com/rancher/wrangler/pkg/relatedresource"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
	k8score "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
)

func Start(ctx context.Context, kubeconfig string, ns string, masterurl string, threadiness int) error {
	logrus.Infof("Starting Terraform Controller, namespace: %s", ns)

	cfg, err := clientcmd.BuildConfigFromFlags(masterurl, kubeconfig)
	if err != nil {
		return fmt.Errorf("error building kubeconfig: %s", err.Error())
	}

	tfFactory, err := terraformcontroller.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		return fmt.Errorf("error building controller controllers: %s", err.Error())
	}

	coreFactory, err := core.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		return fmt.Errorf("error building core controllers: %s", err.Error())
	}

	rbacFactory, err := rbac.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		return fmt.Errorf("error building rbac controllers: %s", err.Error())
	}

	batchFactory, err := batch.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		return fmt.Errorf("error building rbac controllers: %s", err.Error())
	}

	Register(ctx,
		tfFactory.Terraformcontroller().V1().Module(),
		tfFactory.Terraformcontroller().V1().State(),
		tfFactory.Terraformcontroller().V1().Execution(),
		rbacFactory.Rbac().V1().ClusterRole(),
		rbacFactory.Rbac().V1().ClusterRoleBinding(),
		coreFactory.Core().V1().Secret(),
		coreFactory.Core().V1().ConfigMap(),
		coreFactory.Core().V1().ServiceAccount(),
		batchFactory.Batch().V1().Job(),
	)

	return start.All(ctx, threadiness, tfFactory, coreFactory, rbacFactory, batchFactory)
}

func Register(
	ctx context.Context,
	modules tfv1.ModuleController,
	states tfv1.StateController,
	executions tfv1.ExecutionController,
	clusterRoles rbacv1.ClusterRoleController,
	clusterRoleBindings rbacv1.ClusterRoleBindingController,
	secrets corev1.SecretController,
	configMaps corev1.ConfigMapController,
	serviceAccounts corev1.ServiceAccountController,
	jobs batchv1.JobController,
) {
	// watch for modules
	relatedresource.Watch(ctx, "state-module-watch",
		func(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
			var statesFound []string
			var result []relatedresource.Key
			stateList, err := states.List(namespace, metaV1.ListOptions{})
			if err != nil {
				return nil, err
			}
			for _, state := range stateList.Items {
				if _, ok := obj.(*v1.Module); ok {
					module := obj.(*v1.Module)
					if state.Spec.ModuleName == module.Name {
						statesFound = append(statesFound, state.Name)
					}
				}
			}
			if len(statesFound) > 0 {
				for _, foundState := range statesFound {
					result = append(result, relatedresource.NewKey(namespace, foundState))
				}
				return result, nil
			}
			return nil, nil
		},
		states,
		modules)
	// watch configs and secrets
	relatedresource.Watch(ctx, "state-config-secret-watch",
		func(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
			var statesFound []string
			var result []relatedresource.Key

			stateList, err := states.List(namespace, metaV1.ListOptions{})
			if err != nil {
				return nil, err
			}
			for _, state := range stateList.Items {
				if _, ok := obj.(*k8score.ConfigMap); ok {
					for _, envCm := range state.Spec.Variables.EnvConfigName {
						config := obj.(*k8score.ConfigMap)
						if envCm == config.Name {
							statesFound = append(statesFound, state.Name)
						}
					}
					for _, configmap := range state.Spec.Variables.ConfigNames {
						config := obj.(*k8score.ConfigMap)
						if configmap == config.Name {
							statesFound = append(statesFound, state.Name)
						}
					}
				}
				if _, ok := obj.(*k8score.Secret); ok {
					for _, envSecret := range state.Spec.Variables.EnvSecretNames {
						secret := obj.(*k8score.Secret)
						if envSecret == secret.Name {
							statesFound = append(statesFound, state.Name)
						}
					}
					for _, rSecret := range state.Spec.Variables.SecretNames {
						secret := obj.(*k8score.Secret)
						if rSecret == secret.Name {
							statesFound = append(statesFound, state.Name)
						}
					}
				}

			}
			if len(statesFound) > 0 {
				for _, foundState := range statesFound {
					result = append(result, relatedresource.NewKey(namespace, foundState))
				}
				return result, nil
			}
			return nil, nil
		},
		states,
		configMaps,
		secrets)

	stateHandler := state.NewHandler(
		ctx,
		modules,
		states,
		executions,
		clusterRoles,
		clusterRoleBindings,
		secrets,
		configMaps,
		serviceAccounts,
		jobs)
	states.OnChange(ctx, "states-handler", stateHandler.OnChange)
	states.OnRemove(ctx, "states-handler", stateHandler.OnRemove)

	moduleHandler := module.NewHandler(ctx, modules, secrets)
	modules.OnChange(ctx, "modules-handler", moduleHandler.OnChange)
	modules.OnRemove(ctx, "modules-handler", moduleHandler.OnRemove)

	executionHandler := execution.NewHandler(ctx, executions, states, modules)
	executions.OnChange(ctx, "execution-handler", executionHandler.OnChange)
	executions.OnRemove(ctx, "execution-handler", executionHandler.OnRemove)
}
