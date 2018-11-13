package deploy

import (
	"github.com/rancher/kerraform/pkg/digest"
	"github.com/rancher/kerraform/types/apis/kerraform.cattle.io/v1"
	"github.com/rancher/norman/pkg/apply"
	corev1 "k8s.io/api/core/v1"
)

type Input struct {
	Module     *v1.Module
	Executions map[string]*v1.Execution
	Configs    []*corev1.ConfigMap
	Secrets    []*corev1.Secret
}

func Deploy(execution *v1.Execution, input *Input) (string, error) {

}

func Remove(execution *v1.Execution) error {

}

func populate(execution *v1.Execution, input *Input) (*apply.Data, error) {
	content := input.Module.Status.Content
	contentHash := input.Module.Status.ContentHash
	secretHash, secretData := secretData(input)
	configData := configData(input)
	executionRuns := executionRuns(execution, input)

}

func executionRuns(execution *v1.Execution, input *Input) map[string]string {
	result := map[string]string{}
	for dataName, executionName := range execution.Spec.Data {
		result[dataName] = input.Executions[dataName].Status.ExecutionRunName
	}
	return result
}

func configData(input *Input) map[string]string {
	result := map[string]string{}

	for _, config := range input.Configs {
		for k, v := range config.Data {
			result[k] = v
		}
	}

	return result
}

func secretData(input *Input) (string, map[string][]byte) {
	result := map[string][]byte{}
	stringData := map[string]string{}

	for _, secret := range input.Secrets {
		for k, v := range secret.Data {
			result[k] = v
			stringData[k] = string(v)
		}
	}

	return digest.SHA256Map(stringData), result
}

func configHash(configs []*corev1.ConfigMap, secrets []*corev1.Secret) map[string]string {
	vars := map[string]string{}

	for _, config := range configs {
		for k, v := range config.Data {
			vars[k] = v
		}
	}

	for _, secret := range secrets {
		for k, v := range secret.Data {
			vars[k] = string(v)
		}
	}

	return vars
}
