package e2e

import (
	"context"
	"time"

	tfv1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	tf "github.com/rancher/terraform-controller/pkg/generated/clientset/versioned/typed/terraformcontroller.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/crd"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type E2E struct {
	ctx        context.Context
	cs         *kubernetes.Clientset
	cfg        *rest.Config
	kubeconfig string
	namespace  string
	moduleURL  string
	crds       []crd.CRD
}

func NewE2E(namespace, kubeconfig, module string, crdsNames []string) *E2E {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	var crds = make([]crd.CRD, len(crdsNames))
	for k, v := range crdsNames {
		crds[k] = crd.FromGV(tfv1.SchemeGroupVersion, v)
	}

	return &E2E{
		ctx:        signals.SetupSignalHandler(context.Background()),
		cs:         cs,
		cfg:        cfg,
		kubeconfig: kubeconfig,
		namespace:  namespace,
		moduleURL:  module,
		crds:       crds,
	}
}

func (e *E2E) createState() error {
	cs, err := tf.NewForConfig(e.cfg)
	if err != nil {
		return err
	}

	_, err = cs.States(e.namespace).Create(e.ctx, e.getState(), v12.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (e *E2E) createModule() error {
	cs, err := tf.NewForConfig(e.cfg)
	if err != nil {
		return err
	}

	_, err = cs.Modules(e.namespace).Create(e.ctx, e.getModule(), v12.CreateOptions{})
	if err != nil {
		return err
	}

	err = wait.Poll(time.Second, 45*time.Second, func() (bool, error) {
		module, err := cs.Modules(e.namespace).Get(e.ctx, e.generateModuleName(), v12.GetOptions{})
		if err == nil && module.Status.ContentHash != "" {
			return true, nil
		}

		if errors.IsNotFound(err) {
			return false, nil
		}

		logrus.Printf("Waiting for Module to be processed by terraform-controller: %+v\n", err)
		return false, err
	})

	if err != nil {
		return err
	}

	return nil
}

func (e *E2E) createVariables() error {
	_, err := e.cs.CoreV1().Secrets(e.namespace).Create(e.ctx, e.getSecret(), v12.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = e.cs.CoreV1().Secrets(e.namespace).Create(e.ctx, e.getSecretEnv(), v12.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = e.cs.CoreV1().ConfigMaps(e.namespace).Create(e.ctx, e.getConfigMap(), v12.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = e.cs.CoreV1().ConfigMaps(e.namespace).Create(e.ctx, e.getConfigMapEnv(), v12.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (e *E2E) getState() *tfv1.State {
	return &tfv1.State{
		ObjectMeta: v12.ObjectMeta{
			Name:      e.generateStateName(),
			Namespace: e.namespace,
		},
		Spec: tfv1.StateSpec{
			ModuleName:      e.generateModuleName(),
			AutoConfirm:     true,
			DestroyOnDelete: true,
			Image:           "terraform-controller-executor:e2e",
			Variables: tfv1.Variables{
				ConfigNames:    []string{e.generateConfigMapName()},
				EnvConfigName:  []string{e.generateConfigMapEnvName()},
				SecretNames:    []string{e.generateSecretName()},
				EnvSecretNames: []string{e.generateSecretEnvName()},
			},
		},
	}
}

func (e *E2E) generateStateName() string {
	return e.namespace + "-state"
}

func (e *E2E) generateModuleName() string {
	return e.namespace + "-module"
}

func (e *E2E) generateSecretName() string {
	return e.namespace + "-secret"
}

func (e *E2E) generateSecretEnvName() string {
	return e.namespace + "-secret-env"
}

func (e *E2E) getSecret() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      e.generateSecretName(),
			Namespace: e.namespace,
		},
		Type: "opaque",
		StringData: map[string]string{
			"test_secret": e.namespace,
		},
	}
}

func (e *E2E) getSecretEnv() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      e.generateSecretEnvName(),
			Namespace: e.namespace,
		},
		Type: "opaque",
		StringData: map[string]string{
			"test_secret_env": e.namespace,
		},
	}
}

func (e *E2E) generateConfigMapName() string {
	return e.namespace + "-config-map"
}

func (e *E2E) generateConfigMapEnvName() string {
	return e.namespace + "-config-map-env"
}

func (e *E2E) getConfigMap() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name:      e.generateConfigMapName(),
			Namespace: e.namespace,
		},
		Data: map[string]string{
			"test_config_map": e.namespace,
		},
	}
}

func (e *E2E) getConfigMapEnv() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name:      e.generateConfigMapEnvName(),
			Namespace: e.namespace,
		},
		Data: map[string]string{
			"test_config_map_env": e.namespace,
		},
	}
}

func (e *E2E) getModule() *tfv1.Module {
	return &tfv1.Module{
		ObjectMeta: v12.ObjectMeta{
			Name:      e.generateModuleName(),
			Namespace: e.namespace,
		},
		Spec: tfv1.ModuleSpec{
			ModuleContent: tfv1.ModuleContent{
				Git: tfv1.GitLocation{
					URL: e.moduleURL,
				},
			},
		},
	}
}
