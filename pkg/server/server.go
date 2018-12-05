package server

import (
	"context"

	"github.com/ibuildthecloud/terraform-operator/pkg/controllers/module"
	corev1 "github.com/ibuildthecloud/terraform-operator/types/apis/core/v1"
	corev1client "github.com/ibuildthecloud/terraform-operator/types/apis/core/v1"
	"github.com/ibuildthecloud/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"github.com/rancher/norman"
	"github.com/rancher/norman/types"
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
			corev1.Factory,
		},

		LeaderLockNamespace: ns,

		MasterControllers: []norman.ControllerRegister{
			func(ctx context.Context) error {
				return module.Register(ctx, ns, v1.From(ctx), corev1client.From(ctx))
			},
		},
	}
}
