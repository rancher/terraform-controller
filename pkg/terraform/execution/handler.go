package execution

import (
	"context"

	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	"github.com/sirupsen/logrus"
)

func NewHandler(ctx context.Context, executions tfv1.ExecutionController, states tfv1.StateController, modules tfv1.ModuleController) *handler {
	return &handler{
		ctx:        ctx,
		states:     states,
		executions: executions,
		modules:    modules,
	}
}

type handler struct {
	ctx        context.Context
	executions tfv1.ExecutionController
	states     tfv1.StateController
	modules    tfv1.ModuleController
}

func (h *handler) OnChange(key string, execution *v1.Execution) (*v1.Execution, error) {
	logrus.Debug("Execution On Change")
	if execution == nil {
		return nil, nil
	}

	h.states.Enqueue(execution.Namespace, execution.Labels["state"])

	return execution, nil
}

func (h *handler) OnRemove(key string, execution *v1.Execution) (*v1.Execution, error) {
	logrus.Debug("Execution On Remove")

	return execution, nil
}
