//go:build e2e

package classic

import (
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("install classic fullstack")
	builder.WithLabel("name", "classic")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakube.New(
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithClassicFullstackSpec(&dynatracev1beta1.HostInjectSpec{}),
	)

	// check if oneAgent pods startup and report as ready
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	return builder.Feature()
}
