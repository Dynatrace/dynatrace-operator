package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLogMonitoringWithoutK8SMonitoring(t *testing.T) {
	t.Run("no error if logMonitoring is enabled with activegate with k8s-monitoring", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				APIURL:        testApiUrl,
				LogMonitoring: &logmonitoring.Spec{},
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			},
		}
		assertAllowed(t, dk)
	})
	t.Run("error if logMonitoring is enabled with automatic k8s monitoring feature flag but no activegate with k8s-monitoring", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				APIURL:        testApiUrl,
				LogMonitoring: &logmonitoring.Spec{},
			},
		}
		assertAllowedWithWarnings(t, 1, dk)
	})
	t.Run("error if logMonitoring is enabled with activegate with k8s-monitoring but automatic-kubernetes-monitoring disables", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynakube.AnnotationFeatureAutomaticK8sApiMonitoring: "false",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:        testApiUrl,
				LogMonitoring: &logmonitoring.Spec{},
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
				Templates: dynakube.TemplatesSpec{
					LogMonitoring: &logmonitoring.TemplateSpec{
						ImageRef: image.Ref{
							Repository: "repo/image",
							Tag:        "version",
						},
					},
				},
			},
		}
		assertAllowedWithWarnings(t, 2, dk)
	})
	t.Run("error if logMonitoring is enabled without activegate with k8s-monitoring and automatic-kubernetes-monitoring disabled", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynakube.AnnotationFeatureAutomaticK8sApiMonitoring: "false",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:        testApiUrl,
				LogMonitoring: &logmonitoring.Spec{},
				Templates: dynakube.TemplatesSpec{
					LogMonitoring: &logmonitoring.TemplateSpec{
						ImageRef: image.Ref{
							Repository: "repo/image",
							Tag:        "version",
						},
					},
				},
			},
		})
	})
}

func TestIgnoredLogMonitoringTemplate(t *testing.T) {
	t.Run("no warning if logMonitoring template section is empty", func(t *testing.T) {
		dk := createStandaloneLogMonitoringDynakube(testName, testApiUrl, "")
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		dk.Spec.Templates.LogMonitoring = nil
		assertAllowedWithWarnings(t, 1, dk)
	})
	t.Run("warning if logMonitoring template section is not empty", func(t *testing.T) {
		dk := createStandaloneLogMonitoringDynakube(testName, testApiUrl, "something")
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		assertAllowedWithWarnings(t, 2, dk)
	})
}

func createStandaloneLogMonitoringDynakube(name, apiUrl, nodeSelector string) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:        apiUrl,
			LogMonitoring: &logmonitoring.Spec{},
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				},
			},
			Templates: dynakube.TemplatesSpec{
				LogMonitoring: &logmonitoring.TemplateSpec{
					ImageRef: image.Ref{
						Repository: "repo/image",
						Tag:        "version",
					},
				},
			},
		},
	}

	if nodeSelector != "" {
		if dk.Spec.Templates.LogMonitoring == nil {
			dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{}
		}

		dk.Spec.Templates.LogMonitoring.NodeSelector = map[string]string{"node": nodeSelector}
	}

	return dk
}

func TestMissingLogMonitoringImage(t *testing.T) {
	t.Run("both standalone log monitoring and image ref set", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						LogMonitoring: &logmonitoring.TemplateSpec{
							ImageRef: image.Ref{
								Repository: "repo/image",
								Tag:        "version",
							},
						},
					},
				},
			})
	})

	t.Run("standalone log monitoring but missing image", func(t *testing.T) {
		assertDenied(t,
			[]string{errorLogMonitoringMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
				},
			})
	})

	t.Run("standalone log monitoring and only image repository set", func(t *testing.T) {
		assertDenied(t,
			[]string{errorLogMonitoringMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
					Templates: dynakube.TemplatesSpec{
						LogMonitoring: &logmonitoring.TemplateSpec{
							ImageRef: image.Ref{
								Repository: "repo/image",
							},
						},
					},
				},
			})
	})

	t.Run("kspm enabled and only image repository tag", func(t *testing.T) {
		assertDenied(t,
			[]string{errorLogMonitoringMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
					Templates: dynakube.TemplatesSpec{
						LogMonitoring: &logmonitoring.TemplateSpec{
							ImageRef: image.Ref{
								Tag: "version",
							},
						},
					},
				},
			})
	})
}
