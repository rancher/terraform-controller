package execution

import (
	"fmt"

	"github.com/rancher/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"github.com/sirupsen/logrus"

	coreV1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
)

func (e *executionLifecycle) gatherInput(obj *v1.Execution) (*Input, bool, error) {
	var (
		ns   = obj.Namespace
		spec = obj.Spec
	)
	logrus.Debug("Getting module")
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
	logrus.Debug("Getting secrets")

	secrets, ok, err := e.getSecrets(ns, spec)
	if !ok || err != nil {
		return nil, false, err
	}
	logrus.Debug("Getting configs")

	configs, ok, err := e.getConfigs(ns, spec)
	if !ok || err != nil {
		return nil, false, err
	}
	logrus.Debug("Getting executions")

	executions, ok, err := e.getExecutions(ns, spec)
	if !ok || err != nil {
		return nil, false, err
	}

	envVars, ok, err := e.getEnvVars(ns, spec)
	if !ok || err != nil {
		return nil, false, err
	}

	return &Input{
		Configs:    configs,
		EnvVars:    envVars,
		Executions: executions,
		Image:      spec.Image,
		Module:     mod,
		Secrets:    secrets,
	}, true, nil
}

func (e *executionLifecycle) getSecrets(ns string, spec v1.ExecutionSpec) ([]*coreV1.Secret, bool, error) {
	var secrets []*coreV1.Secret

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

func (e *executionLifecycle) getConfigs(ns string, spec v1.ExecutionSpec) ([]*coreV1.ConfigMap, bool, error) {
	var configMaps []*coreV1.ConfigMap

	for _, name := range spec.Variables.ConfigNames {
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

func (e *executionLifecycle) getEnvVars(ns string, spec v1.ExecutionSpec) ([]coreV1.EnvVar, bool, error) {
	result := []coreV1.EnvVar{}

	for _, name := range spec.Variables.EnvSecretNames {
		secret, err := e.secretsLister.Get(ns, name)
		if k8sError.IsNotFound(err) {
			return result, false, nil
		} else if err != nil {
			return result, false, err
		}

		for k, v := range secret.Data {
			e := coreV1.EnvVar{
				Name:  k,
				Value: string(v),
			}
			result = append(result, e)
		}
	}

	for _, name := range spec.Variables.EnvConfigName {
		config, err := e.configMapLister.Get(ns, name)
		if k8sError.IsNotFound(err) {
			return result, false, nil
		} else if err != nil {
			return result, false, err
		}

		for k, v := range config.Data {
			e := coreV1.EnvVar{
				Name:  k,
				Value: v,
			}
			result = append(result, e)
		}
	}
	return result, true, nil
}
