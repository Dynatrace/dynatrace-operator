package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtensionExecutionControllerImage(t *testing.T) {
	t.Run(`the image specified`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Extensions: &dynakube.ExtensionsSpec{},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
						},
					},
				},
			})
	})

	t.Run(`missing tag`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorExtensionExecutionControllerImageNotSpecified},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Extensions: &dynakube.ExtensionsSpec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
							},
						},
					},
				},
			})
	})

	t.Run(`missing repository`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorExtensionExecutionControllerImageNotSpecified},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Extensions: &dynakube.ExtensionsSpec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: image.Ref{
								Tag: "b",
							},
						},
					},
				},
			})
	})

	t.Run(`image not specified`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorExtensionExecutionControllerImageNotSpecified},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Extensions: &dynakube.ExtensionsSpec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
				},
			})
	})
}

func TestExtensionExecutionControllerPVCSettings(t *testing.T) {
	t.Run(`EphemeralVolume disabled and PVC specified`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Extensions: &dynakube.ExtensionsSpec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
							UseEphemeralVolume:    false,
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimSpec{},
						},
					},
				},
			})
	})
	t.Run(`EphemeralVolume enabled and no PVC specified`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Extensions: &dynakube.ExtensionsSpec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
							UseEphemeralVolume: true,
						},
					},
				},
			})
	})
	t.Run(`EphemeralVolume enabled and PVC specified`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorExtensionExecutionControllerInvalidPVCConfiguration},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Extensions: &dynakube.ExtensionsSpec{},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
							UseEphemeralVolume:    true,
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimSpec{},
						},
					},
				},
			})
	})
}

func TestWarnIfmultiplyDKwithExtensionsEnabled(t *testing.T) {
	imgRef := image.Ref{
		Repository: "a",
		Tag:        "b",
	}
	// we want to exclude AG resources warning.
	agSpec := activegate.Spec{
		Capabilities: []activegate.CapabilityDisplayName{
			activegate.KubeMonCapability.DisplayName,
		},
		CapabilityProperties: activegate.CapabilityProperties{
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("1"),
				},
			},
		},
	}
	dk1 := &dynakube.DynaKube{
		ObjectMeta: defaultDynakubeObjectMeta,
		Spec: dynakube.DynaKubeSpec{
			APIURL:     testApiUrl,
			Extensions: &dynakube.ExtensionsSpec{},
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
					ImageRef: imgRef,
				},
			},
			ActiveGate: agSpec,
		},
	}

	t.Run("no warning different ApiUrls", func(t *testing.T) {
		dk2 := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "second",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:     "https://f.q.d.n/123",
				Extensions: &dynakube.ExtensionsSpec{},
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
						ImageRef: imgRef,
					},
				},
				ActiveGate: agSpec,
			},
		}
		assertAllowedWithoutWarnings(t, dk1, dk2)
	})
	t.Run("warning same ApiUrls", func(t *testing.T) {
		dk2 := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "second",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:     testApiUrl,
				Extensions: &dynakube.ExtensionsSpec{},
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
						ImageRef: imgRef,
					},
				},
				ActiveGate: agSpec,
			},
		}
		warnings, err := assertAllowed(t, dk1, dk2)
		require.NoError(t, err)
		require.Len(t, warnings, 1)

		expected := fmt.Sprintf(warningConflictingApiUrlForExtensions, dk2.Name)
		assert.Equal(t, expected, warnings[0])
	})

	t.Run("no warning same ApiUrls and for second dk: extensions feature is disabled", func(t *testing.T) {
		dk2 := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "second",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:     testApiUrl,
				Extensions: nil,
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
						ImageRef: imgRef,
					},
				},
				ActiveGate: agSpec,
			},
		}
		assertAllowedWithoutWarnings(t, dk1, dk2)
	})
}
