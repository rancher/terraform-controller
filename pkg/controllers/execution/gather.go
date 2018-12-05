package execution

import (
	"github.com/ibuildthecloud/terraform-operator/pkg/controllers/execution/deploy"
	"github.com/ibuildthecloud/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (e *executionLifecycle) gatherInput(obj *v1.Execution) (*deploy.Input, bool, error) {
	var (
		ns   = obj.Namespace
		spec = obj.Spec
	)

	mod, err := e.moduleLister.Get(ns, spec.ModuleName)
	if errors.IsNotFound(err) {
		return nil, false, nil
	} else if err != nil {
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

	return &deploy.Input{
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
		if errors.IsNotFound(err) {
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
		if errors.IsNotFound(err) {
			return configMaps, false, nil
		} else if err != nil {
			return configMaps, false, err
		}

		configMaps = append(configMaps, configMap)
	}

	return configMaps, true, nil
}

func (e *executionLifecycle) getExecutions(ns string, spec v1.ExecutionSpec) (map[string]*v1.Execution, bool, error) {
	result := map[string]*v1.Execution{}
	for dataName, execName := range spec.Data {
		execution, err := e.executionLister.Get(ns, execName)
		if errors.IsNotFound(err) {
			return result, false, nil
		} else if err != nil {
			return result, false, err
		}

		if execution.Status.ExecutionRunName == "" {
			return result, false, nil
		}

		result[dataName] = execution
	}

	return result, true, nil
}
