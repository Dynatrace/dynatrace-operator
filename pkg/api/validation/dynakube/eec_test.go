package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

func TestExtensionExecutionControllerImage(t *testing.T) {
	t.Run(`the image specified`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Extensions: dynakube.ExtensionsSpec{
						Enabled: true,
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: common.ImageRefSpec{
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
					APIURL: testApiUrl,
					Extensions: dynakube.ExtensionsSpec{
						Enabled: true,
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: common.ImageRefSpec{
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
					APIURL: testApiUrl,
					Extensions: dynakube.ExtensionsSpec{
						Enabled: true,
					},
					Templates: dynakube.TemplatesSpec{
						ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
							ImageRef: common.ImageRefSpec{
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
					APIURL: testApiUrl,
					Extensions: dynakube.ExtensionsSpec{
						Enabled: true,
					},
				},
			})
	})
}
