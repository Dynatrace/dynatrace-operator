package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImageFieldHasTenantImage(t *testing.T) {
	testTenantUrl := "https://beepboop.dev.dynatracelabs.com"

	t.Run("image fields are a malformed ref", func(t *testing.T) {
		expectedMessage := strings.Join([]string{
			fmt.Sprintf(errorUnparsableImageRef, "ActiveGate"),
			fmt.Sprintf(errorUnparsableImageRef, "OneAgent"),
		}, ";")

		assertDenied(t, []string{expectedMessage}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: &dynakube.HostInjectSpec{
						Image: "BOOM",
					},
				},
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{
						Image: "BOOM",
					},
				},
			},
		})
	})
	t.Run("image fields are using tenant repos", func(t *testing.T) {
		expectedMessage := strings.Join([]string{
			fmt.Sprintf(errorUsingTenantImageAsCustom, "ActiveGate"),
			fmt.Sprintf(errorUsingTenantImageAsCustom, "OneAgent"),
		}, ";")

		assertDenied(t, []string{expectedMessage}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: &dynakube.HostInjectSpec{
						Image: testTenantUrl + "/linux/oneagent:latest",
					},
				},
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{
						Image: testTenantUrl + "/linux/activegate:latest",
					},
				},
			},
		})
	})

	t.Run("valid image fields", func(t *testing.T) {
		testRegistryUrl := "my.images.com"
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: &dynakube.HostInjectSpec{
						Image: testRegistryUrl + "/linux/oneagent:latest",
					},
				},
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{
						Image: testRegistryUrl + "/linux/activegate:latest",
					},
				},
			},
		})
	})
}
