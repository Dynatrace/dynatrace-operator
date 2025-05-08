package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/kspm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTooManyAGReplicas(t *testing.T) {
	t.Run("activegate with 1 (per default) replica and kspm enabled", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Kspm:   &kspm.Spec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
							ImageRef: image.Ref{
								Repository: "repo/image",
								Tag:        "version",
							},
						},
					},
				},
			})
	})

	t.Run("activegate with more than 1 replica and kspm enabled", func(t *testing.T) {
		activeGate := activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.KubeMonCapability.DisplayName,
			},
		}
		replicas := int32(3)

		activeGate.Replicas = &replicas
		assertDenied(t,
			[]string{errorTooManyAGReplicas},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Kspm:       &kspm.Spec{},
					ActiveGate: activeGate,
					Templates: dynakube.TemplatesSpec{
						KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
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

func TestMissingKSPMDependency(t *testing.T) {
	t.Run("both kspm and kubemon enabled", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Kspm:   &kspm.Spec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
							ImageRef: image.Ref{
								Repository: "repo/image",
								Tag:        "version",
							},
						},
					},
				},
			})
	})

	t.Run("missing kubemon but kspm enabled", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Kspm:       &kspm.Spec{},
					ActiveGate: activegate.Spec{},
					Templates: dynakube.TemplatesSpec{
						KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
							ImageRef: image.Ref{
								Repository: "repo/image",
								Tag:        "version",
							},
						},
					},
				},
			})
	})

	t.Run("both kspm and kubemon enabled, automatic k8s monitoring disabled", func(t *testing.T) {
		assertAllowedWithWarnings(t, 2,
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						exp.AGAutomaticK8sApiMonitoringKey: "false",
					},
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Kspm:   &kspm.Spec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
							ImageRef: image.Ref{
								Repository: "repo/image",
								Tag:        "version",
							},
						},
					},
				},
			})
	})

	t.Run("missing kubemon, automatic k8s monitoring disabled, but kspm enabled", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1,
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						exp.AGAutomaticK8sApiMonitoringKey: "false",
					},
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Kspm:       &kspm.Spec{},
					ActiveGate: activegate.Spec{},
					Templates: dynakube.TemplatesSpec{
						KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
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

func TestMissingKSPMImage(t *testing.T) {
	t.Run("both kspm enabled and image ref set", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Kspm:   &kspm.Spec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
							ImageRef: image.Ref{
								Repository: "repo/image",
								Tag:        "version",
							},
						},
					},
				},
			})
	})

	t.Run("kspm enabled but missing image", func(t *testing.T) {
		assertDenied(t,
			[]string{errorKSPMMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Kspm:   &kspm.Spec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
				},
			})
	})

	t.Run("kspm enabled and only image repository set", func(t *testing.T) {
		assertDenied(t,
			[]string{errorKSPMMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Kspm:   &kspm.Spec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
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
			[]string{errorKSPMMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Kspm:   &kspm.Spec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
							ImageRef: image.Ref{
								Tag: "version",
							},
						},
					},
				},
			})
	})
}
