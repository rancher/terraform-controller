package execution

import (
	"fmt"

	"github.com/ibuildthecloud/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
)

func (e *executionLifecycle) gatherInput(obj *v1.Execution) (*Input, bool, error) {
	var (
		ns   = obj.Namespace
		spec = obj.Spec
	)

	mod, err := e.moduleLister.Get(ns, spec.ModuleName)
	if err != nil {
		if k8sError.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	if mod.Status.ContentHash == "" {
		return nil, false, nil
	}

	secrets, ok, err := e.getSecrets(ns, spec)
	if !ok || err != nil {
		return nil, false, err
	}

	configs, ok, err := e.getConfigs(ns, spec)
	if !ok || err != nil {
		return nil, false, err
	}

	executions, ok, err := e.getExecutions(ns, spec)
	if !ok || err != nil {
		return nil, false, err
	}

	return &Input{
		Module:     mod,
		Executions: executions,
		Configs:    configs,
		Secrets:    secrets,
	}, true, nil
}

func (e *executionLifecycle) getSecrets(ns string, spec v1.ExecutionSpec) ([]*corev1.Secret, bool, error) {
	var secrets []*corev1.Secret

	for _, name := range spec.Variables.SecretNames {
		secret, err := e.secretsLister.Get(ns, name)
		if k8sError.IsNotFound(err) {
			return secrets, false, nil
		} else if err != nil {
			return secrets, false, err
		}

		secrets = append(secrets, secret)
	}

	return secrets, true, nil
}

func (e *executionLifecycle) getConfigs(ns string, spec v1.ExecutionSpec) ([]*corev1.ConfigMap, bool, error) {
	var configMaps []*corev1.ConfigMap

	for _, name := range spec.Variables.SecretNames {
		configMap, err := e.configMapLister.Get(ns, name)
		if k8sError.IsNotFound(err) {
			return configMaps, false, nil
		} else if err != nil {
			return configMaps, false, err
		}

		configMaps = append(configMaps, configMap)
	}

	return configMaps, true, nil
}

func (e *executionLifecycle) getExecutions(ns string, spec v1.ExecutionSpec) (map[string]string, bool, error) {
	result := map[string]string{}
	for dataName, execName := range spec.Data {
		execution, err := e.executionLister.Get(ns, execName)
		if k8sError.IsNotFound(err) {
			return result, false, nil
		} else if err != nil {
			return result, false, err
		}

		if execution.Status.ExecutionRunName == "" {
			return result, false, fmt.Errorf("referenced execution %v does not have any runs", execName)
		}

		result[dataName] = execution.Status.ExecutionRunName
	}

	return result, true, nil
}
