package dynakube

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNameStartsWithDigit(t *testing.T) {
	t.Run(`dynakube name starts with digit`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorNoDNS1053Label}, &dynatracev1beta1.DynaKube{
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

func TestNameTooLong(t *testing.T) {
	t.Run(`normal name`, func(t *testing.T) {
		assertAllowedResponse(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: "https://tenantid.doma.in/api",
			},
		})
	})
	t.Run(`name too long`, func(t *testing.T) {
		n := dynatracev1beta1.MaxNameLength + 2
		letters := make([]rune, n)
		for i := 0; i < n; i++ {
			letters[i] = 'a'
		}
		errorMessage := fmt.Sprintf(errorNameTooLong, dynatracev1beta1.MaxNameLength)
		assertDeniedResponse(t, []string{errorMessage}, &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: string(letters),
			},
		})
	})
}
