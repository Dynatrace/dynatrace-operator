package edgeconnect

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNameTooLong(t *testing.T) {
	t.Run(`normal name`, func(t *testing.T) {
		assertAllowedResponse(t, &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name: "edgeconnect",
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "id." + allowedSuffix[0],
			},
		})
	})
	t.Run(`name too long`, func(t *testing.T) {
		n := edgeconnect.MaxNameLength + 2
		letters := make([]rune, n)
		for i := 0; i < n; i++ {
			letters[i] = 'a'
		}
		errorMessage := fmt.Sprintf(errorNameTooLong, edgeconnect.MaxNameLength)
		assertDeniedResponse(t, []string{errorMessage}, &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name: string(letters),
			},
		})
	})
}
