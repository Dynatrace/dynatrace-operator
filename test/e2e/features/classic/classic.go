//go:build e2e

package classic

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
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

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	return builder.Feature()
}
