package dynakube

import (
	"fmt"
	"strings"
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNameStartsWithDigit(t *testing.T) {
	t.Run(`dynakube name starts with digit`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorNoDNS1053Label}, &dynatracev1beta2.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "1dynakube",
			},
		})
		assertAllowedResponse(t, &dynatracev1beta2.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynatracev1beta2.DynaKubeSpec{
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
			crNameLength: dynatracev1beta2.MaxNameLength - 1,
			allow:        true,
		},
		{
			name:         "max",
			crNameLength: dynatracev1beta2.MaxNameLength,
			allow:        true,
		},
		{
			name:         "max + 1 ",
			crNameLength: dynatracev1beta2.MaxNameLength + 1,
			allow:        false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			dk := &dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.Repeat("a", test.crNameLength),
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: "https://tenantid.doma.in/api",
				},
			}
			if test.allow {
				assertAllowedResponse(t, dk)
			} else {
				errorMessage := fmt.Sprintf(errorNameTooLong, dynatracev1beta2.MaxNameLength)
				assertDeniedResponse(t, []string{errorMessage}, dk)
			}
		})
	}
}
