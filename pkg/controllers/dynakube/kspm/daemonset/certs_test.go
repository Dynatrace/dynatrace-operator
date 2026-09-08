// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
)

func getDynaKubeWithCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	dk := dynakube.DynaKube{}
	dk.ActiveGate().TLSSecretName = "test"
	dk.ActiveGate().Capabilities = []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}

	return dk
}

func getDynaKubeWithAutomaticCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	dk := dynakube.DynaKube{}
	dk.ActiveGate().Capabilities = []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}

	return dk
}

func getDynaKubeWithKubemonCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true")

	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			KubernetesMonitoring: &kubemon.Spec{
				TLSCertsRef: &kubemon.TLSCertsRef{
					SecretName: "test",
				},
			},
		},
	}

	return dk
}

func getDynaKubeWithKubemonAutomaticCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true")

	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			KubernetesMonitoring: &kubemon.Spec{},
		},
	}

	return dk
}
