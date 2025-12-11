package validation

import (
	"fmt"
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
		{"max with ID length 1", 40, 1, true},
		{"max with ID length 2", 40, 2, true},
		{"max with ID length 3", 40, 3, true},
		{"max with ID length 4", 39, 4, true},
		{"max with ID length 5", 38, 5, true},
		{"max with ID length 6", 37, 6, true},
		{"max with ID length 7", 36, 7, true},
		{"max with ID length 8", 35, 8, true},
		{"max + 1 with ID length 1", 41, 1, false},
		{"max + 1 with ID length 2", 41, 2, false},
		{"max + 1 with ID length 3", 41, 3, false},
		{"max + 1 with ID length 4", 40, 4, false},
		{"max + 1 with ID length 5", 39, 5, false},
		{"max + 1 with ID length 6", 38, 6, false},
		{"max + 1 with ID length 7", 37, 7, false},
		{"max + 1 with ID length 8", 36, 8, false},
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
				dk.Spec.Templates.SQLExtensionExecutor.ImageRef.Repository = "repo"
				dk.Spec.Templates.SQLExtensionExecutor.ImageRef.Tag = "tag"
			}

			if test.allow {
				assertAllowed(t, dk)
			} else {
				msg := fmt.Sprintf(errorNameTooLong, dynakube.MaxNameLength)
				if test.idNameLength > 0 {
					maxLength := maxNameLengthForSQLExecutor(dk)
					msg = fmt.Sprintf(errorNameTooLong, maxLength)
					if maxLength < dynakube.MaxNameLength {
						msg += sqlExecutorTooLongSuffix
					}
				}

				assertDenied(t, []string{msg}, dk)
			}
		})
	}
}
