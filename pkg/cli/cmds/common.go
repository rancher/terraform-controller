package cmds

import (
	"context"

	"github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io"
	v1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/batch"
	batchv1 "github.com/rancher/wrangler/pkg/generated/controllers/batch/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	corev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/resolvehome"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
)

type controllers struct {
	modules    v1.ModuleController
	states     v1.StateController
	executions v1.ExecutionController
	configMaps corev1.ConfigMapController
	secrets    corev1.SecretController
	jobs       batchv1.JobController
}

const (
	terraState = "tfstate"
	terraKey   = "tfstateSecretSuffix"
)

var controllerCache *controllers

func getControllers(kc, ns string) (*controllers, error) {
	if controllerCache != nil {
		return controllerCache, nil
	}

	kubeconfig, err := resolvehome.Resolve(kc)

	if err != nil {
		kubeconfig = kc
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	tfFactory, err := terraformcontroller.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		logrus.Fatalf("Error building terraform controllers: %s", err.Error())
	}

	coreFactory, err := core.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		logrus.Fatalf("Error building core controllers: %s", err.Error())
	}

	batchFactory, err := batch.NewFactoryFromConfigWithNamespace(cfg, ns)
	if err != nil {
		logrus.Fatalf("Error building batch controllers: %s", err.Error())
	}

	controllers := &controllers{
		modules:    tfFactory.Terraformcontroller().V1().Module(),
		states:     tfFactory.Terraformcontroller().V1().State(),
		executions: tfFactory.Terraformcontroller().V1().Execution(),
		configMaps: coreFactory.Core().V1().ConfigMap(),
		secrets:    coreFactory.Core().V1().Secret(),
		jobs:       batchFactory.Batch().V1().Job(),
	}

	controllerCache = controllers

	ctx := signals.SetupSignalHandler(context.Background())
	if err := start.All(ctx, 1, tfFactory, coreFactory, batchFactory); err != nil {
		logrus.Fatalf("Error starting: %s", err.Error())
	}

	return controllers, nil
}
