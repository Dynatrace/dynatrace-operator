package validation

import (
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		idNameLength int
		allow        bool
	}

	testCases := []testCase{
		{"normal length", 10, 0, true},
		{"max - 1", dynakube.MaxNameLength - 1, 0, true},
		{"max", dynakube.MaxNameLength, 0, true},
		{"max + 1", dynakube.MaxNameLength + 1, 0, false},
		{"normal length with DB", 10, 8, true},
		{"max - 1 with DB", 24, 8, true},
		{"max with ID length 1", 32, 1, true},
		{"max with ID length 2", 31, 2, true},
		{"max with ID length 3", 30, 3, true},
		{"max with ID length 4", 29, 4, true},
		{"max with ID length 5", 28, 5, true},
		{"max with ID length 6", 27, 6, true},
		{"max with ID length 7", 26, 7, true},
		{"max with ID length 8", 25, 8, true},
		{"max + 1 with DB", 26, 8, false},
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
			if test.idNameLength > 0 {
				dk.Spec.Extensions = &extensions.Spec{
					Databases: []extensions.DatabaseSpec{{ID: strings.Repeat("b", test.idNameLength)}},
				}
				dk.Spec.Templates.ExtensionExecutionController.ImageRef.Repository = "repo"
				dk.Spec.Templates.ExtensionExecutionController.ImageRef.Tag = "tag"
				dk.Spec.Templates.DatabaseExecutor.ImageRef.Repository = "repo"
				dk.Spec.Templates.DatabaseExecutor.ImageRef.Tag = "tag"
			}

			if test.allow {
				assertAllowed(t, dk)
			} else {
				assertDenied(t, []string{"The length limit for the name of a DynaKube is"}, dk)
			}
		})
	}
}
