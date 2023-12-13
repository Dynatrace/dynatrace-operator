//go:build e2e

package classic

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// # ClassicFullStack deployment
//
// Verification of classic-fullstack deployment. Sample application Deployment is
// installed and restarted to check if OneAgent is injected and can communicate
// with the *Dynatrace Cluster*.
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
