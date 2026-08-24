// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
)

func TestKubemonMutualExclusiveCustomPropertiesValue(t *testing.T) {
	t.Run("no custom properties", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL:               testAPIURL,
				KubernetesMonitoring: &kubemon.Spec{},
			},
		})
	})

	t.Run("only value", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				KubernetesMonitoring: &kubemon.Spec{
					CustomProperties: &value.Source{Value: "inline"},
				},
			},
		})
	})

	t.Run("only valueFrom", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				KubernetesMonitoring: &kubemon.Spec{
					CustomProperties: &value.Source{ValueFrom: "secret"},
				},
			},
		})
	})

	t.Run("value and valueFrom", func(t *testing.T) {
		t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true")
		assertDenied(t, []string{errorMutualExclusiveKubernetesMonitoringValue}, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				KubernetesMonitoring: &kubemon.Spec{
					CustomProperties: &value.Source{
						Value:     "inline",
						ValueFrom: "secret",
					},
				},
			},
		})
	})
}

func TestMutualExclusiveKubernetesMonitoring(t *testing.T) {
	t.Run("only kubernetesMonitoring", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL:               testAPIURL,
				KubernetesMonitoring: &kubemon.Spec{},
			},
		})
	})

	t.Run("kubernetesMonitoring and ActiveGate capability", func(t *testing.T) {
		t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true")
		assertDenied(t, []string{errorMutualExclusiveKubernetesMonitoring}, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
				KubernetesMonitoring: &kubemon.Spec{},
			},
		})
	})
}
