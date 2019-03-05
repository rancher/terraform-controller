package server

import (
	"context"

	"github.com/rancher/norman"
	"github.com/rancher/norman/types"
	"github.com/rancher/terraform-operator/pkg/controllers/execution"
	"github.com/rancher/terraform-operator/pkg/controllers/module"
	batchv1 "github.com/rancher/terraform-operator/types/apis/batch/v1"
	"github.com/rancher/terraform-operator/types/apis/client"
	corev1 "github.com/rancher/terraform-operator/types/apis/core/v1"
	rbacv1 "github.com/rancher/terraform-operator/types/apis/rbac.authorization.k8s.io/v1"
	"github.com/rancher/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
)

func Config(ns string) *norman.Config {
	return &norman.Config{
		Name: "terraform-operator",
		Schemas: []*types.Schemas{
			v1.Schemas,
		},

		CRDs: map[*types.APIVersion][]string{
			&v1.APIVersion: {
				v1.ModuleGroupVersionKind.Kind,
				v1.ExecutionGroupVersionKind.Kind,
				v1.ExecutionRunGroupVersionKind.Kind,
			},
		},

		Clients: []norman.ClientFactory{
			v1.Factory,
			batchv1.Factory,
			corev1.Factory,
			rbacv1.Factory,
		},

		LeaderLockNamespace: ns,

		GlobalSetup: client.BuildContext,

		MasterControllers: []norman.ControllerRegister{
			client.Register(func(ctx context.Context, client *client.MasterClient) error {
				return module.Register(ctx, ns, client)
			}),
			client.Register(func(ctx context.Context, client *client.MasterClient) error {
				return execution.Register(ctx, ns, client)
			}),
		},
	}
}
