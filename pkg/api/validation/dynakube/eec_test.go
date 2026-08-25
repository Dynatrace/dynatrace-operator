// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtensionExecutionControllerImage(t *testing.T) {
	t.Run("the image specified", func(t *testing.T) {
		assertAllowed(t, withDatabasesExtension(&dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: extensions.ExecutionControllerSpec{
						ImageRef: image.Ref{
							Repository: "a",
							Tag:        "b",
						},
					},
					OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
						ImageRef: image.Ref{
							Repository: "a",
							Tag:        "b",
						},
					},
					SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
						ImageRef: image.Ref{
							Repository: "a",
							Tag:        "b",
						},
					},
				},
			},
		}))
	})

	t.Run("missing tag", func(t *testing.T) {
		assertDenied(t,
			[]string{errorExtensionExecutionControllerImageNotSpecified},
			withDatabasesExtension(&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: extensions.ExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
							},
						},
						SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
						},
					},
				},
			}))
	})

	t.Run("missing repository", func(t *testing.T) {
		assertDenied(t,
			[]string{errorExtensionExecutionControllerImageNotSpecified},
			withDatabasesExtension(&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: extensions.ExecutionControllerSpec{
							ImageRef: image.Ref{
								Tag: "b",
							},
						},
						SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
						},
					},
				},
			}))
	})

	t.Run("image not specified", func(t *testing.T) {
		assertDenied(t,
			[]string{errorExtensionExecutionControllerImageNotSpecified},
			withDatabasesExtension(&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
						},
					},
				},
			}))
	})

}

func TestExtensionControllerImageNotRequired(t *testing.T) {
	newDK := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
					CapabilityProperties: activegate.CapabilityProperties{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
						},
					},
				},
				Extensions: &extensions.Spec{
					Databases: []extensions.DatabaseSpec{{ID: "test"}},
				},
			},
		}
	}
	t.Run("image not required when public registry is used", func(t *testing.T) {
		dk := newDK()
		dk.Annotations = map[string]string{exp.UsePublicRegistryKey: "true"}
		assertAllowedWithoutWarnings(t, dk)
	})
	t.Run("image not required when platform token is present", func(t *testing.T) {
		dk := newDK()
		assertAllowedWithoutWarnings(t, dk, platformTokenSecret())
	})
}

func TestExtensionExecutionControllerPVCSettings(t *testing.T) {
	t.Run("EphemeralVolume disabled and PVC specified", func(t *testing.T) {
		assertAllowed(t, withDatabasesExtension(&dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: extensions.ExecutionControllerSpec{
						ImageRef: image.Ref{
							Repository: "a",
							Tag:        "b",
						},
						UseEphemeralVolume:    new(false),
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimSpec{},
					},
					OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
						ImageRef: image.Ref{
							Repository: "a",
							Tag:        "b",
						},
					},
					SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
						ImageRef: image.Ref{
							Repository: "repo",
							Tag:        "tag",
						},
					},
				},
			},
		}))
	})

	t.Run("EphemeralVolume enabled and no PVC specified", func(t *testing.T) {
		assertAllowed(t, withDatabasesExtension(&dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: extensions.ExecutionControllerSpec{
						ImageRef: image.Ref{
							Repository: "a",
							Tag:        "b",
						},
						UseEphemeralVolume: new(true),
					},
					OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
						ImageRef: image.Ref{
							Repository: "a",
							Tag:        "b",
						},
					},
					SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
						ImageRef: image.Ref{
							Repository: "repo",
							Tag:        "tag",
						},
					},
				},
			},
		}))
	})

	t.Run("EphemeralVolume enabled and PVC specified", func(t *testing.T) {
		assertDenied(t,
			[]string{errorExtensionExecutionControllerInvalidPVCConfiguration},
			withDatabasesExtension(&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: extensions.ExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
							UseEphemeralVolume:    new(true),
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimSpec{},
						},
						SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
							ImageRef: image.Ref{
								Repository: "repo",
								Tag:        "tag",
							},
						},
					},
				},
			}))
	})

}

func TestWarnIfmultipleDKwithExtensionsEnabled(t *testing.T) {
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
			APIURL: testAPIURL,
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: extensions.ExecutionControllerSpec{
					ImageRef: imgRef,
				},
				OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
					ImageRef: image.Ref{
						Repository: "otc/repo",
						Tag:        "otc-tag",
					},
				},
				SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
					ImageRef: image.Ref{
						Repository: "repo",
						Tag:        "tag",
					},
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
				APIURL: "https://f.q.d.n/123",
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: extensions.ExecutionControllerSpec{
						ImageRef: imgRef,
					},
					SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
						ImageRef: imgRef,
					},
				},
				ActiveGate: agSpec,
			},
		}
		assertAllowedWithWarnings(t, 1, withDatabasesExtension(dk1), withDatabasesExtension(dk2))
	})

	t.Run("warning same ApiUrls", func(t *testing.T) {
		dk2 := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "second",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: extensions.ExecutionControllerSpec{
						ImageRef: imgRef,
					},
					SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
						ImageRef: imgRef,
					},
				},
				ActiveGate: agSpec,
			},
		}
		warnings, err := assertAllowed(t, withDatabasesExtension(dk1), withDatabasesExtension(dk2))
		require.NoError(t, err)
		require.Len(t, warnings, 2)

		expected := fmt.Sprintf(warningConflictingAPIURLForExtensions, dk2.Name)
		assert.Contains(t, warnings, expected)
	})

	t.Run("no warning same ApiUrls and for second dk: extensions feature is disabled", func(t *testing.T) {
		dk2 := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "second",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:     testAPIURL,
				Extensions: nil,
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: extensions.ExecutionControllerSpec{
						ImageRef: imgRef,
					},
					SQLExtensionExecutor: extensions.DatabaseExecutorSpec{
						ImageRef: imgRef,
					},
				},
				ActiveGate: agSpec,
			},
		}
		assertAllowedWithWarnings(t, 1, withDatabasesExtension(dk1), dk2)
	})

}
