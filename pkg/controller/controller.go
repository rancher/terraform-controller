package controller

import (
	"context"

	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/controller/execution"
	"github.com/rancher/terraform-controller/pkg/controller/module"
	"github.com/rancher/terraform-controller/pkg/controller/state"
	"github.com/rancher/terraform-controller/pkg/types"
	"github.com/rancher/wrangler/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	k8score "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Start(ctx context.Context,
	c *types.Controllers,
	ns string) error {

	logrus.Infof("Starting Terraform Controller, namespace: %s", ns)
	// watch for modules
	relatedresource.Watch(ctx, "state-module-watch",
		func(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
			var statesFound []string
			var result []relatedresource.Key
			stateList, err := c.State.List(namespace, metaV1.ListOptions{})
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
		c.State,
		c.Module)
	// watch configs and secrets
	relatedresource.Watch(ctx, "state-config-secret-watch",
		func(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
			var statesFound []string
			var result []relatedresource.Key

			stateList, err := c.State.List(namespace, metaV1.ListOptions{})
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
		c.State,
		c.ConfigMap,
		c.Secret)

	stateHandler := state.NewHandler(
		ctx,
		c.Module,
		c.State,
		c.Execution,
		c.ClusterRole,
		c.ClusterRoleBinding,
		c.Secret,
		c.ConfigMap,
		c.ServiceAccount,
		c.Job)
	c.State.OnChange(ctx, "states-handler", stateHandler.OnChange)
	c.State.OnRemove(ctx, "states-handler", stateHandler.OnRemove)

	moduleHandler := module.NewHandler(ctx, c.Module, c.Secret)
	c.Module.OnChange(ctx, "modules-handler", moduleHandler.OnChange)
	c.Module.OnRemove(ctx, "modules-handler", moduleHandler.OnRemove)

	executionHandler := execution.NewHandler(ctx, c.Execution, c.State, c.Module)
	c.Execution.OnChange(ctx, "execution-handler", executionHandler.OnChange)
	c.Execution.OnRemove(ctx, "execution-handler", executionHandler.OnRemove)

	return nil
}
