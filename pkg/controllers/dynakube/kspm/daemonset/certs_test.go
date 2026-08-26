// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
)

func getDynaKubeWithCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	dk := dynakube.DynaKube{}
	dk.Spec.ActiveGate = &activegate.Spec{
		TLSSecretName: "test",
		Capabilities:  []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName},
	}

	return dk
}

func getDynaKubeWithAutomaticCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	dk := dynakube.DynaKube{}
	dk.Spec.ActiveGate = &activegate.Spec{
		Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName},
	}

	return dk
}
