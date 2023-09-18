package dynakube

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNameStartsWithDigit(t *testing.T) {
	t.Run(`dynakube name starts with digit`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorDigitInName}, &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "1dynakube",
			},
		})
		assertAllowedResponse(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: "https://tenantid.doma.in/api",
			},
		})
	})
}
