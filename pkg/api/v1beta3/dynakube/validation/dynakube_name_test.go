package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNameStartsWithDigit(t *testing.T) {
	t.Run(`dynakube name starts with digit`, func(t *testing.T) {
		assertDenied(t, []string{errorNoDNS1053Label}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "1dynakube",
			},
		})
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://tenantid.doma.in/api",
			},
		})
	})
}

func TestNameTooLong(t *testing.T) {
	type testCase struct {
		name         string
		crNameLength int
		allow        bool
	}

	testCases := []testCase{
		{
			name:         "normal length",
			crNameLength: 10,
			allow:        true,
		},
		{
			name:         "max - 1 ",
			crNameLength: dynakube.MaxNameLength - 1,
			allow:        true,
		},
		{
			name:         "max",
			crNameLength: dynakube.MaxNameLength,
			allow:        true,
		},
		{
			name:         "max + 1 ",
			crNameLength: dynakube.MaxNameLength + 1,
			allow:        false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			dk := &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.Repeat("a", test.crNameLength),
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: "https://tenantid.doma.in/api",
				},
			}
			if test.allow {
				assertAllowed(t, dk)
			} else {
				errorMessage := fmt.Sprintf(errorNameTooLong, dynakube.MaxNameLength)
				assertDenied(t, []string{errorMessage}, dk)
			}
		})
	}
}
