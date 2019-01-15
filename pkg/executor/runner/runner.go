package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ibuildthecloud/terraform-operator/pkg/executor/terraform"
	"github.com/ibuildthecloud/terraform-operator/pkg/executor/writer"
	"github.com/ibuildthecloud/terraform-operator/pkg/git"
	"github.com/ibuildthecloud/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	approvalMessage = `autoConfirm is not set, and the annotation 'approved' is empty,
please review the plan and set the annotation 'approved' to 'yes' if approved
or 'no' if not approved. If set to 'no' the job will exit without making any changes.`
)

type Runner struct {
	Action       string
	Namespace    string
	ExecutionRun *v1.ExecutionRun
	GitAuth      *git.Auth
	K8sClient    *kubernetes.Clientset
	OpClient     *v1.Clients
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
	opClient, err := v1.NewForConfig(*config)
	if err != nil {
		return nil, err
	}

	r.OpClient = v1.NewClientsFromInterface(opClient)

	return &r, nil
}

// TerraformInit runs the terraform init command
func (r *Runner) TerraformInit() (string, error) {
	return terraform.Init()
}

// Create will create resources through terraform. If the execution AutoConfirm flag is
// set it will run 'plan' then 'apply', if the flag is not set 'plan' will run then
// the job will wait for the approved annotation to be set then the job will run 'apply' or exit.
func (r *Runner) Create() (string, error) {
	out, err := terraform.Plan(false)
	if err != nil {
		return "", err
	}
	fmt.Println(out)
	logrus.Info(out)
	err = r.SetExecutionRunStatus("planned")
	if err != nil {
		return "", err
	}

	// We have autoConfirm, run apply
	if r.ExecutionRun.Spec.AutoConfirm {
		return terraform.Apply()
	}

	// Need to wait for approval before running apply
	approval, ok := r.ExecutionRun.Annotations["approved"]
	if !ok || approval == "" {
		logrus.Info(approvalMessage)
		approval, err = r.waitForApproval()
		if err != nil {
			return "", err
		}
	}

	switch strings.ToLower(approval) {
	case "no":
		return "", errors.New("annotation 'approved' set to 'no', no changes applied, exiting job")
	case "yes":
		return terraform.Apply()
	default:
		return "", fmt.Errorf("invalid value set for annotation 'approved': %v", approval)
	}
}

// Destroy will destroy resources through terraform. If the execution AutoConfirm flag is
// set it will run 'destroy', if the flag is not set 'destroy' will run then
// the job will wait for the approved approved to be set then the job will run 'destroy'
// or exit
func (r *Runner) Destroy() (string, error) {
	out, err := terraform.Plan(true)
	if err != nil {
		return "", err
	}

	logrus.Info(out)

	// We have autoConfirm, run apply
	if r.ExecutionRun.Spec.AutoConfirm {
		return terraform.Destroy()
	}

	// Need to wait for approval before running apply
	approval, ok := r.ExecutionRun.Annotations["approved"]
	if !ok || approval == "" {
		logrus.Info(approvalMessage)
		approval, err = r.waitForApproval()
		if err != nil {
			return "", err
		}
	}

	switch strings.ToLower(approval) {
	case "no":
		return "Annotation 'approved' set to 'no', no changes applied. Exiting job.", nil
	case "yes":
		return terraform.Destroy()
	default:
		return "", fmt.Errorf("invalid value set for annotation 'approved': %v", approval)
	}
}

func (r *Runner) SaveOutputs() error {
	output, err := terraform.Output()
	if err != nil {
		return err
	}

	return tryUpdate(func() error {
		run, err := r.OpClient.ExecutionRun.Get(r.ExecutionRun.Namespace, r.ExecutionRun.Name, metaV1.GetOptions{})
		if err != nil {
			return err
		}

		r.ExecutionRun.Status.Outputs = output

		_, err = r.OpClient.ExecutionRun.Update(run)
		if err != nil {
			return err
		}
		// Update runner so we have the fresh version
		r.ExecutionRun = run
		return nil
	})
}

// Populate attempts to grab all resources needed for running
func (r *Runner) Populate() error {
	name := os.Getenv("EXECUTOR_RUN_NAME")
	if name == "" {
		return errors.New("executor run name not set")
	}

	action := os.Getenv("EXECUTOR_ACTION")
	if action == "" {
		return errors.New("action not set")
	}
	r.Action = strings.ToLower(action)

	ns := os.Getenv("EXECUTOR_NAMESPACE")
	if ns == "" {
		return errors.New("namespace not set")
	}
	r.Namespace = ns

	run, err := r.getExecutionRun(ns, name)
	if err != nil {
		return err
	}
	r.ExecutionRun = run

	if r.ExecutionRun.Spec.Content.Git.SecretName != "" {
		gSecret, err := r.getSecret(r.ExecutionRun.Spec.Content.Git.SecretName)
		if err != nil {
			return err
		}
		auth, err := git.FromSecret(gSecret.Data)
		if err != nil {
			return err
		}
		r.GitAuth = &auth
	} else {
		r.GitAuth = &git.Auth{}
	}

	vSecret, err := r.getSecret(r.ExecutionRun.Spec.SecretName)
	if err != nil {
		return err
	}
	r.VarSecret = vSecret

	return nil
}

func (r *Runner) SetExecutionRunStatus(s string) error {
	return tryUpdate(func() error {
		run, err := r.getExecutionRun(r.Namespace, r.ExecutionRun.Name)
		if err != nil {
			return err
		}

		switch s {
		case "planned":
			v1.ExecutionRunConditionPlanned.SetStatus(run, "true")
		case "applied":
			v1.ExecutionRunConditionApplied.SetStatus(run, "true")
		default:
			return fmt.Errorf("unknown execution run status: %v", s)
		}

		run, err = r.OpClient.ExecutionRun.Update(run)
		if err != nil {
			return err
		}
		r.ExecutionRun = run
		return nil
	})
}

func (r *Runner) WriteConfigFile() error {
	config := Config{
		Terraform: Terraform{
			Backend: map[string]*Backend{
				"kubernetes": &Backend{
					Key:            r.ExecutionRun.Spec.ExecutionName,
					Namespace:      r.ExecutionRun.Namespace,
					ServiceAccount: "true",
				},
			},
		},
	}

	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return err
	}

	err = writer.Write(jsonConfig, "/root/module/config.tf.json")
	if err != nil {
		return err
	}

	return nil
}

func (r *Runner) WriteVarFile() error {
	vars, ok := r.VarSecret.Data["varFile"]
	if !ok {
		return fmt.Errorf("no varFile data found in secret %v", r.VarSecret.Name)
	}
	err := writer.Write(vars, fmt.Sprintf("/root/module/%v.auto.tfvars", r.ExecutionRun.Name))
	if err != nil {
		return err
	}
	return nil
}

func (r *Runner) waitForApproval() (string, error) {
	opts := metaV1.ListOptions{}
	watch, err := r.OpClient.ExecutionRun.Watch(opts)
	if err != nil {
		return "", err
	}
	defer watch.Stop()

	for event := range watch.ResultChan() {
		run, ok := event.Object.(*v1.ExecutionRun)
		if !ok {
			return "", errors.New("unexpected type")
		}

		if run.Name != r.ExecutionRun.Name {
			continue
		}

		approval, ok := run.Annotations["approved"]
		if !ok || approval == "" {
			continue
		}

		return approval, nil

	}
	return "", nil
}

func (r *Runner) getExecutionRun(namespace, name string) (*v1.ExecutionRun, error) {
	return r.OpClient.ExecutionRun.Get(namespace, name, metaV1.GetOptions{})
}

func (r *Runner) getSecret(name string) (*coreV1.Secret, error) {
	return r.K8sClient.CoreV1().Secrets(r.ExecutionRun.ObjectMeta.Namespace).Get(name, metaV1.GetOptions{})
}

func tryUpdate(f func() error) error {
	timeout := 100
	for i := 0; i <= 3; i++ {
		err := f()
		if err != nil {
			if k8sError.IsConflict(err) {
				time.Sleep(time.Duration(timeout) * time.Millisecond)
				timeout *= 2
				continue
			}
			return err
		}
	}
	return nil
}
