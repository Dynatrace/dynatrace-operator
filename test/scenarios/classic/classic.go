//go:build e2e

package classic

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func install(t *testing.T) features.Feature {
	builder := features.New("install classic fullstack")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		ClassicFullstack(&dynatracev1beta1.HostInjectSpec{}).
		Build()

	// check if oneAgent pods startup and report as ready
	steps.CreateFeatureEnvironment(builder,
		steps.CreateNamespaceWithoutTeardown(namespace.NewBuilder(testDynakube.Namespace).Build()),
		steps.DeployOperatorViaMake(testDynakube.Namespace, testDynakube.NeedsCSIDriver()),
		steps.CreateDynakube(secretConfig, testDynakube),
	)
	return builder.Feature()
}
