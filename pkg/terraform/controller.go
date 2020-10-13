package terraform

import (
	"context"

	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/terraform/execution"
	"github.com/rancher/terraform-controller/pkg/terraform/module"
	"github.com/rancher/terraform-controller/pkg/terraform/state"
	batchv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/batch/v1"
	corev1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	rbacv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/rbac/v1"
	"github.com/rancher/wrangler/pkg/relatedresource"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
				if _,ok := obj.(*core.ConfigMap); ok {
					for _, envCm := range state.Spec.Variables.EnvConfigName {
						config:= obj.(*core.ConfigMap)
						if envCm == config.Name {
							statesFound = append(statesFound, state.Name)
						}
					}
					for _, configmap := range state.Spec.Variables.ConfigNames {
						config:= obj.(*core.ConfigMap)
						if configmap == config.Name {
							statesFound = append(statesFound, state.Name)
						}
					}
				}
				if _,ok := obj.(*core.Secret); ok {
					for _, envSecret := range state.Spec.Variables.EnvSecretNames {
						secret:= obj.(*core.Secret)
						if envSecret == secret.Name {
							statesFound = append(statesFound, state.Name)
						}
					}
					for _, rSecret := range state.Spec.Variables.SecretNames {
						secret:= obj.(*core.Secret)
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
