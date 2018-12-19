package runner

import (
	"errors"
	"os"

	"github.com/ibuildthecloud/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Runner struct {
	Namespace    string
	K8sClient    *kubernetes.Clientset
	OpClient     *v1.Clients
	ExecutionRun *v1.ExecutionRun
	GitSecret    *coreV1.Secret
	VarSecret    *coreV1.Secret
}

// NewRunner returns a runner with the k8s clients populated
func NewRunner(config *rest.Config) (*Runner, error) {
	var r Runner
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	r.K8sClient = client

	opClient, err := v1.NewClients(*config)
	if err != nil {
		return nil, err
	}

	r.OpClient = opClient

	return &r, nil
}

// Populate attempts to grab all resources needed for running
func (r *Runner) Populate() error {
	name := os.Getenv("EXECUTOR_RUN_NAME")
	if name == "" {
		return errors.New("no run name set")
	}

	ns := os.Getenv("EXECUTOR_NAMESPACE")
	if ns == "" {
		return errors.New("no namespace set")
	}
	r.Namespace = ns

	run, err := r.getExecutionRun(name)
	if err != nil {
		return err
	}
	r.ExecutionRun = run

	gSecret, err := r.getSecret(r.ExecutionRun.Spec.Content.Git.SecretName)
	if err != nil {
		return err
	}
	r.GitSecret = gSecret

	vSecret, err := r.getSecret(name)
	if err != nil {
		return err
	}
	r.VarSecret = vSecret

	return nil
}

func (r *Runner) getExecutionRun(name string) (*v1.ExecutionRun, error) {
	return r.OpClient.ExecutionRun.Get(r.Namespace, name, metaV1.GetOptions{})
}

func (r *Runner) getSecret(name string) (*coreV1.Secret, error) {
	return r.K8sClient.CoreV1().Secrets(r.Namespace).Get(name, metaV1.GetOptions{})
}
