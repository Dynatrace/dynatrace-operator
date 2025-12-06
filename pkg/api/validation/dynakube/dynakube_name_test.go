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
		withDBExt    bool
		allow        bool
	}

	testCases := []testCase{
		{"normal length", 10, false, true},
		{"normal length with DB", 10, true, true},
		{"max - 1", dynakube.MaxNameLength - 1, false, true},
		{"max - 1 with DB", dynakube.MaxNameLength - len(extensions.SQLExecutorInfix) - 2, true, true},
		{"max", dynakube.MaxNameLength, false, true},
		{"max with DB", dynakube.MaxNameLength - len(extensions.SQLExecutorInfix) - 1, true, true},
		{"max + 1", dynakube.MaxNameLength + 1, false, false},
		{"max + 1 with DB", dynakube.MaxNameLength - len(extensions.SQLExecutorInfix), true, false},
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
			if test.withDBExt {
				dk.Spec.Extensions = &extensions.Spec{
					Databases: []extensions.DatabaseSpec{{ID: "test"}},
				}
				dk.Spec.Templates.ExtensionExecutionController.ImageRef.Repository = "repo"
				dk.Spec.Templates.ExtensionExecutionController.ImageRef.Tag = "tag"
				dk.Spec.Templates.DatabaseExecutor.ImageRef.Repository = "repo"
				dk.Spec.Templates.DatabaseExecutor.ImageRef.Tag = "tag"
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
