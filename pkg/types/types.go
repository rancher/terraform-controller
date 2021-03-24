package types

import (
	"github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io"
	v1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/coordination.k8s.io"
	coordv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/coordination.k8s.io/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/batch"
	batchv1 "github.com/rancher/wrangler/pkg/generated/controllers/batch/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	corev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/rbac"
	rbacv1 "github.com/rancher/wrangler/pkg/generated/controllers/rbac/v1"
)

type Controllers struct {
	Module             v1.ModuleController
	State              v1.StateController
	Execution          v1.ExecutionController
	ClusterRole        rbacv1.ClusterRoleController
	ClusterRoleBinding rbacv1.ClusterRoleBindingController
	Secret             corev1.SecretController
	ConfigMap          corev1.ConfigMapController
	ServiceAccount     corev1.ServiceAccountController
	Job                batchv1.JobController
	Coordination       coordv1.LeaseController
}

type Factories struct {
	Tf    *terraformcontroller.Factory
	Core  *core.Factory
	Rbac  *rbac.Factory
	Batch *batch.Factory
	Lease *coordination.Factory
}
