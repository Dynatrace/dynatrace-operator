package validation

import (
	"context"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
)

func TestHasApiUrl(t *testing.T) {
	dk := &dynakube.DynaKube{}
	assert.Equal(t, errorNoApiUrl, NoApiUrl(context.Background(), nil, dk))

	dk.Spec.APIURL = testApiUrl
	assert.Empty(t, NoApiUrl(context.Background(), nil, dk))

	t.Run(`happy path`, func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://tenantid.doma.in/api",
			},
		})
	})
	t.Run(`valid API URL (no domain)`, func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://...tenantid/api",
			},
		})
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://my-in-cluster-activegate/e/<tenant>/api",
			},
		})
	})
	t.Run(`missing API URL`, func(t *testing.T) {
		assertDenied(t, []string{errorNoApiUrl}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "",
			},
		})
	})
	t.Run(`example API URL`, func(t *testing.T) {
		assertDenied(t, []string{errorNoApiUrl}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: ExampleApiUrl,
			},
		})
	})
	t.Run(`invalid API URL (without /api suffix)`, func(t *testing.T) {
		assertDenied(t, []string{errorInvalidApiUrl}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: strings.TrimSuffix(ExampleApiUrl, "/api"),
			},
		})
	})
	t.Run(`invalid API URL (not a Dynatrace environment)`, func(t *testing.T) {
		assertDenied(t, []string{errorInvalidApiUrl}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://www.google.com",
			},
		})
	})
	t.Run(`invalid API URL (empty tenant ID)`, func(t *testing.T) {
		assertDenied(t, []string{errorInvalidApiUrl}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "/api",
			},
		})
	})
	t.Run(`third gen API URL`, func(t *testing.T) {
		assertDenied(t, []string{errorThirdGenApiUrl}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://tenantid.doma.apps.in/api",
			},
		})
	})
}
