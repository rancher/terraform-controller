package server

import (
	"context"

	"github.com/rancher/kerraform/pkg/controllers/module"
	corev1 "github.com/rancher/kerraform/types/apis/core/v1"
	corev1client "github.com/rancher/kerraform/types/apis/core/v1"
	"github.com/rancher/kerraform/types/apis/kerraform.cattle.io/v1"
	"github.com/rancher/norman"
	"github.com/rancher/norman/types"
)

func Config(ns string) *norman.Config {
	return &norman.Config{
		Name: "kerraform",
		Schemas: []*types.Schemas{
			v1.Schemas,
		},

		CRDs: map[*types.APIVersion][]string{
			&v1.APIVersion: {
				v1.ModuleGroupVersionKind.Kind,
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
