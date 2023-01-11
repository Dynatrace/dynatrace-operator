package operator

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

const (
	Name      = "dynatrace-operator"
	Namespace = "dynatrace"
)

func Get(ctx context.Context, resource *resources.Resources) (appsv1.Deployment, error) {
	return deployment.NewQuery(ctx, resource, client.ObjectKey{
		Name:      Name,
		Namespace: Namespace,
	}).Get()
}
