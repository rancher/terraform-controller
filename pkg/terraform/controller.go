package terraform

import (
	"context"

	"github.com/rancher/terraform-controller/pkg/terraform/state"

	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/terraform/module"
	batchv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/batch/v1"
	corev1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	rbacv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/rbac/v1"
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
	states.OnChange(ctx, "executions-handler", stateHandler.OnChange)
	states.OnRemove(ctx, "executions-handler", stateHandler.OnRemove)

	moduleHandler := module.NewHandler(ctx, modules, secrets)
	modules.OnChange(ctx, "modules-handler", moduleHandler.OnChange)
	modules.OnRemove(ctx, "modules-handler", moduleHandler.OnRemove)
}
