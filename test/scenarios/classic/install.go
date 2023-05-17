//go:build e2e

package classic

import (
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func install(t *testing.T) features.Feature {
	builder := features.New("install classic fullstack")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		ClassicFullstack(&dynatracev1.HostInjectSpec{}).
		Build()

	// check if oneAgent pods startup and report as ready
	assess.InstallDynatraceWithTeardown(builder, &secretConfig, testDynakube)

	return builder.Feature()
}
