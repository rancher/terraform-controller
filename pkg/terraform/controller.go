package terraform

import (
	"context"
	batchv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/batch/v1"
	corev1 "github.com/rancher/terraform-controller/pkg/generated/controllers/core/v1"
	rbacv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/rbac/v1"
	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/terraform/execution"
	"github.com/rancher/terraform-controller/pkg/terraform/module"
)

func Register(
	ctx context.Context,
	executions tfv1.ExecutionController,
	modules tfv1.ModuleController,
	executionRuns tfv1.ExecutionRunController,

	clusterRoles rbacv1.ClusterRoleController,
	clusterRoleBindings rbacv1.ClusterRoleBindingController,
	secrets corev1.SecretController,
	configMaps corev1.ConfigMapController,
	serviceAccounts corev1.ServiceAccountController,
	jobs batchv1.JobController,
) {

	executionHandler := execution.NewHandler(
		ctx,
		modules,
		executions,
		executionRuns,
		clusterRoles,
		clusterRoleBindings,
		secrets,
		configMaps,
		serviceAccounts,
		jobs)
	executions.OnChange(ctx, "executions-handler", executionHandler.OnChange)
	executions.OnRemove(ctx, "executions-handler", executionHandler.OnRemove)

	moduleHandler := module.NewHandler(ctx, modules, secrets)
	modules.OnChange(ctx, "modules-handler", moduleHandler.OnChange)
	modules.OnRemove(ctx, "modules-handler", moduleHandler.OnRemove)
}
