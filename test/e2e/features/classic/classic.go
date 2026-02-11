//go:build e2e

package classic

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
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
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithClassicFullstackSpec(&oneagent.HostInjectSpec{}),
	)

	// check if oneAgent pods startup and report as ready
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
