package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestNameStartsWithDigit(t *testing.T) {
	t.Run("dynakube name starts with digit", func(t *testing.T) {
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
		{"normal length", 10, true},
		{"max - 1", dynakube.MaxNameLength - 1, true},
		{"max", dynakube.MaxNameLength, true},
		{"max + 1", dynakube.MaxNameLength + 1, false},
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
				msg := fmt.Sprintf(errorNameTooLong, dynakube.MaxNameLength)
				assertDenied(t, []string{msg}, dk)
			}
		})
	}
}

func TestInvalidNameErrorMatches(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo.bar",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenantid.doma.in/api",
		},
	}
	upstreamErr := validation.IsDNS1035Label(dk.Name)
	require.Len(t, upstreamErr, 1)
	assertDenied(t, []string{upstreamErr[0]}, dk)
}

func TestNoNameViolationOnTooLongName(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.Repeat("a", 64),
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenantid.doma.in/api",
		},
	}

	_, err := runValidators(dk)
	msg := fmt.Sprintf(errorNameTooLong, dynakube.MaxNameLength)
	require.ErrorContains(t, err, msg)
	assert.NotContains(t, err.Error(), errorNoDNS1053Label)
}
