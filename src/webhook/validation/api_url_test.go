package validation

import (
	"strings"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestHasApiUrl(t *testing.T) {
	instance := &dynatracev1.DynaKube{}
	assert.Equal(t, errorNoApiUrl, NoApiUrl(nil, instance))

	instance.Spec.APIURL = testApiUrl
	assert.Empty(t, NoApiUrl(nil, instance))

	t.Run(`happy path`, func(t *testing.T) {
		assertAllowedResponse(t, &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: "https://tenantid.doma.in/api",
			},
		})
	})
	t.Run(`valid API URL (no domain)`, func(t *testing.T) {
		assertAllowedResponse(t, &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: "https://...tenantid/api",
			},
		})
		assertAllowedResponse(t, &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: "https://my-in-cluster-activegate/e/<tenant>/api",
			},
		})
	})
	t.Run(`missing API URL`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorNoApiUrl}, &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: "",
			},
		})
	})
	t.Run(`example API URL`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorNoApiUrl}, &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: ExampleApiUrl,
			},
		})
	})
	t.Run(`invalid API URL (without /api suffix)`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorInvalidApiUrl}, &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: strings.TrimSuffix(ExampleApiUrl, "/api"),
			},
		})
	})
	t.Run(`invalid API URL (not a Dynatrace environment)`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorInvalidApiUrl}, &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: "https://www.google.com",
			},
		})
	})
	t.Run(`invalid API URL (empty tenant ID)`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorInvalidApiUrl}, &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: "/api",
			},
		})
	})
}
