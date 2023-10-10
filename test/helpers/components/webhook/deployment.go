//go:build e2e

package webhook

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	DeploymentName = webhook.DeploymentName
)

func Get(ctx context.Context, resource *resources.Resources, namespace string) (appsv1.Deployment, error) {
	return deployment.NewQuery(ctx, resource, client.ObjectKey{
		Name:      DeploymentName,
		Namespace: namespace,
	}).Get()
}

func WaitForDeployment(namespace string) features.Func {
	return deployment.WaitFor(DeploymentName, namespace)
}
