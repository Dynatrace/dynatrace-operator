// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testDynakubeName = "dynakube"

func TestExtensionsWithoutKubernetesMonitoringRegistration(t *testing.T) {
	t.Run("no warning if kubernetes monitoring with activegate", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.KubeMonCapability.DisplayName,
			},
		}
		assertAllowed(t, withDatabasesExtension(dk))
	})

	t.Run("no warning if kubernetes monitoring with kubernetesMonitoring with registration", func(t *testing.T) {
		t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true")
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Spec.KubernetesMonitoring = &kubemon.Spec{
			Registration: &kubemon.Registration{},
		}

		warnings, _ := assertAllowed(t, withDatabasesExtension(dk))
		assert.NotContains(t, warnings, warningExtensionsWithoutK8SMonitoring)
	})

	t.Run("warning if kubernetesMonitoring has no registration", func(t *testing.T) {
		t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true")
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Spec.KubernetesMonitoring = &kubemon.Spec{}

		warnings, _ := assertAllowed(t, withDatabasesExtension(dk))
		assert.Contains(t, warnings, warningExtensionsWithoutK8SMonitoring)
	})

	t.Run("warning if kubernetes monitoring is not configured", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		assertAllowedWithWarnings(t, 3, withDatabasesExtension(dk))
	})

	t.Run("warning if activegate kubernetes monitoring has no automatic cluster registration", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Annotations = map[string]string{
			exp.AGAutomaticK8sAPIMonitoringKey: "false",
		}
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.KubeMonCapability.DisplayName,
			},
		}
		assertAllowedWithWarnings(t, 3, withDatabasesExtension(dk))
	})

	t.Run("warning if kubernetes monitoring is not configured or registered", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Annotations = map[string]string{
			exp.AGAutomaticK8sAPIMonitoringKey: "false",
		}
		assertAllowedWithWarnings(t, 3, withDatabasesExtension(dk))
	})
}

func TestExtensionsWithoutOTelCollectorImage(t *testing.T) {
	t.Run("error when image is not specified", func(t *testing.T) {
		assertDenied(t, []string{errorOTelCollectorMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testAPIURL,
					Extensions: &extensions.Spec{},
				},
			})
	})
}

func createStandaloneExtensionsDynakube(name, apiURL string) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: apiURL,
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: extensions.ExecutionControllerSpec{
					ImageRef: image.Ref{
						Repository: "repo/image",
						Tag:        "version",
					},
				},
				OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
					ImageRef: image.Ref{
						Repository: "repo/otel-collector",
						Tag:        "version",
					},
				},
				SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
					ImageRef: image.Ref{
						Repository: "repo/image",
						Tag:        "version",
					},
				},
			},
		},
	}

	return dk
}

func withDatabasesExtension(dk *dynakube.DynaKube) *dynakube.DynaKube {
	dk.Spec.Extensions = &extensions.Spec{Databases: []extensions.DatabaseSpec{{ID: "test"}}}

	return dk
}
