package validation

import (
	"context"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
)

func TestHasApiUrl(t *testing.T) {
	dk := &dynakube.DynaKube{}
	assert.Equal(t, errorNoAPIURL, NoAPIURL(context.Background(), nil, dk))

	dk.Spec.APIURL = testAPIURL
	assert.Empty(t, NoAPIURL(context.Background(), nil, dk))

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
		assertDenied(t, []string{errorNoAPIURL}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "",
			},
		})
	})
	t.Run(`example API URL`, func(t *testing.T) {
		assertDenied(t, []string{errorNoAPIURL}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: ExampleAPIURL,
			},
		})
	})
	t.Run(`invalid API URL (without /api suffix)`, func(t *testing.T) {
		assertDenied(t, []string{errorInvalidAPIURL}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: strings.TrimSuffix(ExampleAPIURL, "/api"),
			},
		})
	})
	t.Run(`invalid API URL (not a Dynatrace environment)`, func(t *testing.T) {
		assertDenied(t, []string{errorInvalidAPIURL}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://www.google.com",
			},
		})
	})
	t.Run(`invalid API URL (empty tenant ID)`, func(t *testing.T) {
		assertDenied(t, []string{errorInvalidAPIURL}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "/api",
			},
		})
	})
	t.Run(`third gen API URL`, func(t *testing.T) {
		assertDenied(t, []string{errorThirdGenAPIURL}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://tenantid.doma.apps.in/api",
			},
		})
	})
	t.Run(`unmutated API URL`, func(t *testing.T) {
		assertUpdateAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					APIURL: "https://tenant.live.dynatrace.com/api",
				},
			},
			&dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					APIURL: "https://tenant.live.dynatrace.com/api",
				},
			},
		)
	})
	t.Run(`mutated API URL`, func(t *testing.T) {
		assertUpdateDenied(t,
			[]string{errorMutatedAPIURL},
			&dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					APIURL: "https://tenant.live.dynatrace.com/api",
				},
			},
			&dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					APIURL: "https://newtenant.live.dynatrace.com/api",
				},
			},
		)
	})
}
