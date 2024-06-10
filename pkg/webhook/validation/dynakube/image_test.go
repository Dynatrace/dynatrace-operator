package dynakube

import (
	"fmt"
	"strings"
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImageFieldHasTenantImage(t *testing.T) {
	testTenantUrl := "https://beepboop.dev.dynatracelabs.com"

	t.Run("image fields are a malformed ref", func(t *testing.T) {
		expectedMessage := strings.Join([]string{
			fmt.Sprintf(errorUnparsableImageRef, "ActiveGate"),
			fmt.Sprintf(errorUnparsableImageRef, "OneAgent"),
		}, ";")

		assertDeniedResponse(t, []string{expectedMessage}, &dynatracev1beta2.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta2.HostInjectSpec{
						Image: "BOOM",
					},
				},
				ActiveGate: dynatracev1beta2.ActiveGateSpec{
					CapabilityProperties: dynatracev1beta2.CapabilityProperties{
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

		assertDeniedResponse(t, []string{expectedMessage}, &dynatracev1beta2.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta2.HostInjectSpec{
						Image: testTenantUrl + "/linux/oneagent:latest",
					},
				},
				ActiveGate: dynatracev1beta2.ActiveGateSpec{
					CapabilityProperties: dynatracev1beta2.CapabilityProperties{
						Image: testTenantUrl + "/linux/activegate:latest",
					},
				},
			},
		})
	})

	t.Run("valid image fields", func(t *testing.T) {
		testRegistryUrl := "https://my.images.com"
		assertAllowedResponse(t, &dynatracev1beta2.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta2.HostInjectSpec{
						Image: testRegistryUrl + "/linux/oneagent:latest",
					},
				},
				ActiveGate: dynatracev1beta2.ActiveGateSpec{
					CapabilityProperties: dynatracev1beta2.CapabilityProperties{
						Image: testRegistryUrl + "/linux/activegate:latest",
					},
				},
			},
		})
	})
}
