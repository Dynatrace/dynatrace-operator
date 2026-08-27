// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testDynakubeName = "dynakube"

func TestExtensionsWithoutK8SMonitoring(t *testing.T) {
	t.Run("no error if activegate with k8s-monitoring", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.KubeMonCapability.DisplayName,
			},
		}
		assertAllowed(t, withDatabasesExtension(dk))
	})

	t.Run("error if no activegate with k8s-monitoring", func(t *testing.T) {
		assertAllowedWithWarnings(t, 3, withDatabasesExtension(createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)))
	})

	t.Run("error if activegate with k8s-monitoring but automatic Kuberenetes API monitoring is disabled", func(t *testing.T) {
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

	t.Run("error if automatic Kuberenetes API monitoring is disabled and without activgate k8s-monitoring", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Annotations = map[string]string{
			exp.AGAutomaticK8sAPIMonitoringKey: "false",
		}
		assertAllowedWithWarnings(t, 3, withDatabasesExtension(dk))
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
