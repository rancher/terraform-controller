package e2e

import (
	"math/rand"
	"os"
	"testing"
	"time"

	tfv1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	tf "github.com/rancher/terraform-controller/pkg/generated/clientset/versioned/typed/terraformcontroller.cattle.io/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	Namespace     = "terraform-controller"
	ModuleURL     = "https://github.com/luthermonson/terraform-controller-test-module"
	TestConfigMap = "test-config-map"
)

var e *E2E

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	kubeconfig := os.Getenv("KUBECONFIG")
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		logrus.Fatal("kubeconfig file does not exist")
	}
	namespace := os.Getenv("Namespace")
	if namespace == "" {
		namespace = Namespace
	}

	e = NewE2E(namespace, kubeconfig, ModuleURL, []string{
		"Module",
		"State",
		"Execution",
	})

	err := e.initialize()
	if err != nil {
		logrus.Fatalf("Failed to initialize: %s", err.Error())
	}
	os.Exit(m.Run())
}

func TestCreateModule(t *testing.T) {
	assert := assert.New(t)
	err := e.createModule()
	if err != nil {
		logrus.Fatalf("Creating terraform module failed: %s", err.Error())
	}

	cs, err := tf.NewForConfig(e.cfg)
	if err != nil {
		logrus.Fatalf("Creating clientset for modules: %s", err.Error())
	}

	err = wait.Poll(time.Second, 15*time.Second, func() (bool, error) {
		var err error
		module, err := cs.Modules(e.namespace).Get(e.ctx, e.generateModuleName(), v13.GetOptions{})

		if err == nil && module.Status.ContentHash != "" {
			return true, nil
		}
		logrus.Printf("Waiting for module to be ready and have a run hash: %+v\n", err)

		return false, err
	})

	assert.Nil(err)
}

func TestCreateVariables(t *testing.T) {
	//assert := assert.New(t)
	err := e.createVariables()
	if err != nil {
		logrus.Fatalf("Creating terraform state failed: %s", err.Error())
	}
}

func TestCreateState(t *testing.T) {

	assert := assert.New(t)
	err := e.createState()
	if err != nil {
		logrus.Fatalf("Creating terraform state failed: %s", err.Error())
	}

	cs, err := tf.NewForConfig(e.cfg)
	if err != nil {
		logrus.Fatalf("Creating clientset for states: %s", err.Error())
	}

	err = wait.Poll(time.Second, 15*time.Second, func() (bool, error) {
		var err error
		state, err := cs.States(e.namespace).Get(e.ctx, e.generateStateName(), v13.GetOptions{})

		if err == nil && state.Status.LastRunHash != "" {
			return true, nil
		}
		logrus.Printf("Waiting for state to be ready and have a run hash: %+v\n", err)

		return false, err
	})

	assert.Nil(err)
}

func TestCreateJobComplete(t *testing.T) {
	assert := assert.New(t)
	var cm *v1.ConfigMap
	err := wait.Poll(time.Second, 30*time.Second, func() (bool, error) {
		var err error
		cm, err = e.cs.CoreV1().ConfigMaps(e.namespace).Get(e.ctx, TestConfigMap, v13.GetOptions{})
		if err == nil {
			return true, nil
		}
		if errors.IsNotFound(err) {
			return false, nil
		}

		logrus.Printf("Waiting for config map creation: %+v\n", err)
		return false, err
	})

	assert.Nil(err)
	assert.Equal(e.namespace, cm.Data["test_config_map"])
	assert.Equal(e.namespace, cm.Data["test_secret"])
}

func TestExecution(t *testing.T) {
	assert := assert.New(t)

	cs, err := tf.NewForConfig(e.cfg)
	if err != nil {
		logrus.Fatalf("Creating clientset for modules: %s", err.Error())
	}

	executions, err := cs.Executions(e.namespace).List(e.ctx, v13.ListOptions{
		LabelSelector: "state=" + e.generateStateName(),
	})

	assert.Nil(err)
	assert.Equal(1, len(executions.Items))

	var jobDeleted = false
	err = wait.Poll(time.Second, 30*time.Second, func() (bool, error) {
		var err error
		_, err = e.cs.BatchV1().Jobs(e.namespace).Get(e.ctx, "job-"+executions.Items[0].Name, v13.GetOptions{})
		if errors.IsNotFound(err) {
			jobDeleted = true
			return true, nil
		}

		return false, nil
	})

	assert.Nil(err)
	exec, err := cs.Executions(e.namespace).Get(e.ctx, executions.Items[0].Name, v13.GetOptions{})
	assert.Nil(err)
	assert.True(jobDeleted)
	assert.NotEmpty(exec.Status.JobLogs)
	assert.True(tfv1.ExecutionRunConditionPlanned.IsTrue(exec))
	assert.True(tfv1.ExecutionRunConditionApplied.IsTrue(exec))
}

func TestTerraState(t *testing.T) {
	assert := assert.New(t)

	ts, err := e.cs.CoreV1().Secrets(e.namespace).List(e.ctx, v13.ListOptions{
		LabelSelector: "tfstateSecretSuffix=" + e.generateStateName(),
	})

	assert.Nil(err)
	assert.Equal(len(ts.Items), 1)
	assert.NotEmpty(ts.Items[0].Data["tfstate"])
	assert.Empty(ts.Items[0].Data["lockInfo"])
}

func TestDeleteState(t *testing.T) {
	assert := assert.New(t)

	cs, err := tf.NewForConfig(e.cfg)
	assert.Nil(err)
	err = cs.States(e.namespace).Delete(e.ctx, e.generateStateName(), v13.DeleteOptions{})
	assert.Nil(err)
}

func TestDeleteJobComplete(t *testing.T) {
	assert := assert.New(t)
	var configMapDeleted = false
	err := wait.Poll(time.Second, 30*time.Second, func() (bool, error) {
		var err error
		_, err = e.cs.CoreV1().ConfigMaps(e.namespace).Get(e.ctx, TestConfigMap, v13.GetOptions{})
		if errors.IsNotFound(err) {
			configMapDeleted = true
			return true, nil
		}

		return false, nil
	})

	assert.Nil(err)
	assert.True(configMapDeleted)
}
