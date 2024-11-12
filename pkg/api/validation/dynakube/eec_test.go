package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	corev1 "k8s.io/api/core/v1"
)

func TestExtensionExecutionControllerImage(t *testing.T) {
	t.Run(`the image specified`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
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
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
							UseEphemeralVolume:    address.Of(false),
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
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
							UseEphemeralVolume: address.Of(true),
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
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: image.Ref{
								Repository: "a",
								Tag:        "b",
							},
							UseEphemeralVolume:    address.Of(true),
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimSpec{},
						},
					},
				},
			})
	})
}
