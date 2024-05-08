package dynakube

import (
	"context"
	"strings"
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/stretchr/testify/assert"
)

func TestHasApiUrl(t *testing.T) {
	instance := &dynatracev1beta2.DynaKube{}
	assert.Equal(t, errorNoApiUrl, NoApiUrl(context.Background(), nil, instance))

	instance.Spec.APIURL = testApiUrl
	assert.Empty(t, NoApiUrl(context.Background(), nil, instance))

	t.Run(`happy path`, func(t *testing.T) {
		assertAllowedResponse(t, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: "https://tenantid.doma.in/api",
			},
		})
	})
	t.Run(`valid API URL (no domain)`, func(t *testing.T) {
		assertAllowedResponse(t, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: "https://...tenantid/api",
			},
		})
		assertAllowedResponse(t, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: "https://my-in-cluster-activegate/e/<tenant>/api",
			},
		})
	})
	t.Run(`missing API URL`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorNoApiUrl}, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: "",
			},
		})
	})
	t.Run(`example API URL`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorNoApiUrl}, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: ExampleApiUrl,
			},
		})
	})
	t.Run(`invalid API URL (without /api suffix)`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorInvalidApiUrl}, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: strings.TrimSuffix(ExampleApiUrl, "/api"),
			},
		})
	})
	t.Run(`invalid API URL (not a Dynatrace environment)`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorInvalidApiUrl}, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: "https://www.google.com",
			},
		})
	})
	t.Run(`invalid API URL (empty tenant ID)`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorInvalidApiUrl}, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: "/api",
			},
		})
	})
	t.Run(`third gen API URL`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorThirdGenApiUrl}, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: "https://tenantid.doma.apps.in/api",
			},
		})
	})
}
