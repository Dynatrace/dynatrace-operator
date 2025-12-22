package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTooManyAGReplicas(t *testing.T) {
	t.Run("activegate with 1 (per default) replica and kspm enabled", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
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
					APIURL:     testAPIURL,
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
					APIURL: testAPIURL,
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
		assertDenied(t, []string{errorKSPMMissingKubemon},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Kspm: &kspm.Spec{
						MappedHostPaths: []string{"/"},
					},
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
		assertDenied(t, []string{errorKSPMMissingKubemon},
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						exp.AGAutomaticK8sAPIMonitoringKey: "false",
					},
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Kspm: &kspm.Spec{
						MappedHostPaths: []string{"/"},
					},
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
		assertDenied(t, []string{errorKSPMMissingKubemon},
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						exp.AGAutomaticK8sAPIMonitoringKey: "false",
					},
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Kspm: &kspm.Spec{
						MappedHostPaths: []string{"/"},
					},
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
					APIURL: testAPIURL,
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
					APIURL: testAPIURL,
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
					APIURL: testAPIURL,
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
					APIURL: testAPIURL,
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

func TestMappedHostPath(t *testing.T) {
	getDynakube := func() dynakube.DynaKube {
		return dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				Kspm:   &kspm.Spec{},
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
					CapabilityProperties: activegate.CapabilityProperties{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{},
						},
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
		}
	}

	t.Run("empty list", func(t *testing.T) {
		dk := getDynakube()
		assertAllowedWithWarnings(t, 1, &dk)
	})

	t.Run("single root path", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.Kspm.MappedHostPaths = []string{"/"}
		assertAllowedWithoutWarnings(t, &dk)
	})

	t.Run("many paths", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.Kspm.MappedHostPaths = []string{"/a", "/b"}
		assertAllowedWithoutWarnings(t, &dk)
	})

	t.Run("many paths with root directory", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.Kspm.MappedHostPaths = []string{"/a", "/b", "/"}
		assertDenied(t, []string{errorKSPMRootHostPath}, &dk)
	})

	t.Run("relative path", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.Kspm.MappedHostPaths = []string{"/a", "b"}
		assertDenied(t, []string{fmt.Sprintf(errorKSPMRelativeHostPath, "b")}, &dk)
	})
}
