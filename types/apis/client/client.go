package client

import (
	"context"

	batchv1 "github.com/ibuildthecloud/terraform-operator/types/apis/batch/v1"
	corev1 "github.com/ibuildthecloud/terraform-operator/types/apis/core/v1"
	rbacv1 "github.com/ibuildthecloud/terraform-operator/types/apis/rbac.authorization.k8s.io/v1"
	"github.com/ibuildthecloud/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"github.com/rancher/norman"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var clientKey struct{}

type MasterClient struct {
	Batch       batchv1.Interface
	Core        corev1.Interface
	K8s         kubernetes.Interface
	LocalConfig *rest.Config
	Operator    v1.Interface
	RBAC        rbacv1.Interface
}

func Store(ctx context.Context, c *MasterClient) context.Context {
	return context.WithValue(ctx, clientKey, c)
}

func From(ctx context.Context) *MasterClient {
	return ctx.Value(clientKey).(*MasterClient)
}

func NewContext(ctx context.Context) *MasterClient {
	server := norman.GetServer(ctx)
	return &MasterClient{
		Batch:       batchv1.From(ctx),
		Core:        corev1.From(ctx),
		K8s:         server.K8sClient,
		LocalConfig: server.LocalConfig,
		Operator:    v1.From(ctx),
		RBAC:        rbacv1.From(ctx),
	}
}

func BuildContext(ctx context.Context) (context.Context, error) {
	return Store(ctx, NewContext(ctx)), nil
}

func Register(f func(context.Context, *MasterClient) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return f(ctx, From(ctx))
	}
}
