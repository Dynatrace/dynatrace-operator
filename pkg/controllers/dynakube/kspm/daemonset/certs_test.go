package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
)

func getDynaKubeWithCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	dk := dynakube.DynaKube{}
	dk.ActiveGate().Spec.TlsSecretName = "test"
	dk.ActiveGate().Capabilities = []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}

	return dk
}
