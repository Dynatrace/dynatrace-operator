//go:build e2e

package classic

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// # ClassicFullStack deployment
//
// Verification of classic-fullstack deployment. Sample application Deployment is
// installed and restarted to check if OneAgent is injected and can communicate
// with the *Dynatrace Cluster*.
func Feature(t *testing.T) features.Feature {
	builder := features.New("classic")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithApiUrl(secretConfig.ApiUrl),
		dynakubeComponents.WithClassicFullstackSpec(&dynakube.HostInjectSpec{}),
	)

	// check if oneAgent pods startup and report as ready
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
