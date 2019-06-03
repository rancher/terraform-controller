package state

import (
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/sirupsen/logrus"

	coreV1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *handler) gatherInput(obj *v1.State) (*Input, bool, error) {
	var (
		ns   = obj.Namespace
		spec = obj.Spec
	)

	mod, err := h.modules.Get(ns, spec.ModuleName, metaV1.GetOptions{})

	if err != nil {
		if k8sError.IsNotFound(err) {
			return nil, false, fmt.Errorf("no module with name %s", spec.ModuleName)
		}
		return nil, false, errors.New("pulling module failed")
	}

	if mod.Status.ContentHash == "" {
		return nil, false, errors.New("module content hash is empty")
	}

	secrets, ok, err := h.getSecrets(ns, spec)
	if !ok || err != nil {
		return nil, false, errors.New("pulling secrets failed")
	}

	configs, ok, err := h.getConfigs(ns, spec)
	if !ok || err != nil {
		return nil, false, errors.New("pulling config maps failed")
	}

	executions, ok, err := h.getExecutions(ns, spec)
	if !ok || err != nil {
		logrus.Debug()
		return nil, false, errors.New("pulling executions failed")
	}

	envVars, ok, err := h.getEnvVars(ns, spec)
	if !ok || err != nil {
		return nil, false, errors.New("pulling environment variables failed")
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

func (h *handler) getSecrets(ns string, spec v1.StateSpec) ([]*coreV1.Secret, bool, error) {
	var secrets []*coreV1.Secret

	for _, name := range spec.Variables.SecretNames {
		secret, err := h.secrets.Get(ns, name, metaV1.GetOptions{})
		if k8sError.IsNotFound(err) {
			return secrets, false, nil
		} else if err != nil {
			return secrets, false, err
		}

		secrets = append(secrets, secret)
	}

	return secrets, true, nil
}

func (h *handler) getConfigs(ns string, spec v1.StateSpec) ([]*coreV1.ConfigMap, bool, error) {
	var configMaps []*coreV1.ConfigMap

	for _, name := range spec.Variables.ConfigNames {
		configMap, err := h.configMaps.Get(ns, name, metaV1.GetOptions{})
		if k8sError.IsNotFound(err) {
			return configMaps, false, nil
		} else if err != nil {
			return configMaps, false, err
		}

		configMaps = append(configMaps, configMap)
	}

	return configMaps, true, nil
}

func (h *handler) getExecutions(ns string, spec v1.StateSpec) (map[string]string, bool, error) {
	result := map[string]string{}
	for dataName, execName := range spec.Data {
		state, err := h.states.Get(ns, execName, metaV1.GetOptions{})
		if k8sError.IsNotFound(err) {
			return result, false, nil
		} else if err != nil {
			return result, false, err
		}

		if state.Status.ExecutionName == "" {
			return result, false, fmt.Errorf("referenced execution %v does not have any runs", execName)
		}

		result[dataName] = state.Status.ExecutionName
	}

	return result, true, nil
}

func (h *handler) getEnvVars(ns string, spec v1.StateSpec) ([]coreV1.EnvVar, bool, error) {
	result := []coreV1.EnvVar{}

	logrus.Debugf("Pulling Vars from Secrets: %d", len(spec.Variables.EnvSecretNames))
	for _, name := range spec.Variables.EnvSecretNames {
		logrus.Debugf("Secret: %s", name)
		secret, err := h.secrets.Get(ns, name, metaV1.GetOptions{})
		if k8sError.IsNotFound(err) {
			logrus.Debugf("Not Found: %s", name)
			return result, false, nil
		} else if err != nil {
			logrus.Debugf("Error: %s", name)
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

	logrus.Debugf("Pulling Env Vars from Config Maps: %d", len(spec.Variables.EnvConfigName))
	for _, name := range spec.Variables.EnvConfigName {
		logrus.Debugf("Env Var: %s", name)
		config, err := h.configMaps.Get(ns, name, metaV1.GetOptions{})
		if k8sError.IsNotFound(err) {
			logrus.Debugf("Not Found: %s", name)
			return result, false, nil
		} else if err != nil {
			logrus.Debugf("Error: %s", name)
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
