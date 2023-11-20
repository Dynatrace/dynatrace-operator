package edgeconnect

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
			crNameLength: edgeconnect.MaxNameLength - 1,
			allow:        true,
		},
		{
			name:         "max",
			crNameLength: edgeconnect.MaxNameLength,
			allow:        true,
		},
		{
			name:         "max + 1 ",
			crNameLength: edgeconnect.MaxNameLength + 1,
			allow:        false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ec := &edgeconnect.EdgeConnect{
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.Repeat("a", test.crNameLength),
				},
				Spec: edgeconnect.EdgeConnectSpec{
					ApiServer: "id." + allowedSuffix[0],
				},
			}
			if test.allow {
				assertAllowedResponse(t, ec)
			} else {
				errorMessage := fmt.Sprintf(errorNameTooLong, edgeconnect.MaxNameLength)
				assertDeniedResponse(t, []string{errorMessage}, ec)
			}
		})
	}
}
