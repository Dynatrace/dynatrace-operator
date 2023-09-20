//go:build e2e

package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func install(t *testing.T) features.Feature {
	builder := features.New("install edgeconnect")

	secretConfig := tenant.GetSingleTenantSecret(t)

	testEdgeConnect := edgeconnect.NewBuilder().
		Build()

	// Register operator install
	setup.CreateFeatureEnvironment(builder,
		setup.CreateNamespaceWithoutTeardown(namespace.NewBuilder(testEdgeConnect.Namespace).Build()),
		setup.DeployOperatorViaMake(false),
		setup.CreateEdgeConnect(secretConfig, testEdgeConnect),
	)
	return builder.Feature()
}
