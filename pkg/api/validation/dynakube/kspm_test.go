package validation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
)

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
				},
			})
	})

	t.Run("missing kubemon but kspm enabled", func(t *testing.T) {
		assertDenied(t,
			[]string{errorKSPMMissingKubemon},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testApiUrl,
					Kspm:       &kspm.Spec{},
					ActiveGate: activegate.Spec{},
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
