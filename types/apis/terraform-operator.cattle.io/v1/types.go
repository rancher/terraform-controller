package v1

import (
	"github.com/rancher/norman/condition"
	"github.com/rancher/norman/types"
	"github.com/rancher/norman/types/factory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	APIVersion = types.APIVersion{
		Group:   "terraform-operator.cattle.io",
		Version: "v1",
		Path:    "/v1-terraform-operator",
	}
	Schemas = factory.
		Schemas(&APIVersion).
		MustImport(&APIVersion, Module{}).
		MustImport(&APIVersion, Execution{}).
		MustImport(&APIVersion, ExecutionRun{})

	ModuleConditionGitUpdated = condition.Cond("GitUpdated")

	ExecutionConditionJobDeployed = condition.Cond("JobDeployed")

	ExecutionRunConditionPlanned = condition.Cond("Planned")
	ExecutionRunConditionApplied = condition.Cond("Applied")
)

type Module struct {
	types.Namespaced

	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModuleSpec   `json:"spec"`
	Status ModuleStatus `json:"status"`
}

type ModuleSpec struct {
	ModuleContent
}

type ModuleContent struct {
	Content map[string]string `json:"content,omitempty"`
	Git     GitLocation       `json:"git,omitempty"`
}

type ModuleStatus struct {
	CheckTime   metav1.Time                  `json:"time,omitempty"`
	GitChecked  *GitLocation                 `json:"gitChecked,omitempty"`
	Content     ModuleContent                `json:"content,omitempty"`
	ContentHash string                       `json:"contentHash,omitempty"`
	Conditions  []condition.GenericCondition `json:"conditions,omitempty"`
}

type GitLocation struct {
	URL             string `json:"url,omitempty"`
	Branch          string `json:"branch,omitempty"`
	Tag             string `json:"tag,omitempty"`
	Commit          string `json:"commit,omitempty"`
	SecretName      string `json:"secretName,omitempty"`
	IntervalSeconds int    `json:"intervalSeconds,omitempty"`
}

type Execution struct {
	types.Namespaced

	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExecutionSpec   `json:"spec"`
	Status ExecutionStatus `json:"status"`
}

type Variables struct {
	SecretNames []string `json:"secretNames,omitempty"`
	ConfigNames []string `json:"configNames,omitempty"`
}

type ExecutionSpec struct {
	Variables   Variables         `json:"variables,omitempty"`
	ModuleName  string            `json:"moduleName,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
	AutoConfirm bool              `json:"autoConfirm,omitempty"`
}

type ExecutionStatus struct {
	ExecutionRunName string `json:"executionRunName,omitempty"`
}

type ExecutionRun struct {
	types.Namespaced

	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExecutionRunSpec   `json:"spec"`
	Status ExecutionRunStatus `json:"status"`
}

type ExecutionRunSpec struct {
	ExecutionName string            `json:"executionName,omitempty"`
	AutoConfirm   bool              `json:"autoConfirm,omitempty"`
	SecretName    string            `json:"secretName,omitempty"`
	ConfigMapName string            `json:"configMapName,omitempty"`
	Content       ModuleContent     `json:"content,omitempty"`
	Data          map[string]string `json:"data,omitempty"`
}

type ExecutionRunStatus struct {
	Conditions    []condition.GenericCondition `json:"conditions,omitempty"`
	JobName       string                       `json:"jobName,omitempty"`
	PlanOutput    string                       `json:"planOutput,omitempty"`
	PlanConfirmed bool                         `json:"planConfirmed,omitempty"`
	ApplyOutput   string                       `json:"applyOutput,omitempty"`
}
