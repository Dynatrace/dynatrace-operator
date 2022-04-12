package validation

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testProxySecret = "proxysecret"

	// invalidPlainTextProxyUrl contains forbidden apostrophe character
	invalidPlainTextProxyUrl = "http://test:test'~!@#$%^*()_-|}{[]\":;?><./pass@proxy-service.dynatrace:3128"

	// validEncodedProxyUrl contains no forbidden characters "http://test:test~!@#$%^*()_-|}{[]\":;?><./pass@proxy-service.dynatrace:3128"
	validEncodedProxyUrl = "http://test:test~!%40%23%5E*()_-%7C%7D%7B%5B%5D%22%3A%3B%3F%3E%3C.%2Fpass@proxy-service.dynatrace:3128"
)

func TestInvalidActiveGateProxy(t *testing.T) {
	t.Run(`valid proxy url`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     validEncodedProxyUrl,
						ValueFrom: "",
					},
				},
			})
	})

	t.Run(`invalid proxy url`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorInvalidActiveGateProxyUrl},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     invalidPlainTextProxyUrl,
						ValueFrom: "",
					},
				},
			})
	})

	t.Run(`valid proxy secret url`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     "",
						ValueFrom: testProxySecret,
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProxySecret,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"proxy": []byte(validEncodedProxyUrl),
				},
			})
	})

	t.Run(`missing proxy secret`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorMissingActiveGateProxySecret},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     "",
						ValueFrom: testProxySecret,
					},
				},
			})
	})

	t.Run(`invalid format of proxy secret`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorInvalidProxySecretFormat},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     "",
						ValueFrom: testProxySecret,
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProxySecret,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"invalid-name": []byte(validEncodedProxyUrl),
				},
			})
	})

	t.Run(`invalid proxy secret url`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorInvalidProxySecretUrl},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     "",
						ValueFrom: testProxySecret,
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProxySecret,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"proxy": []byte(invalidPlainTextProxyUrl),
				},
			})
	})

	t.Run(`invalid proxy secret url - entrypoint.sh`, func(t *testing.T) {
		assert.Equal(t, true, isStringValidForAG("password"))
		assert.Equal(t, true, isStringValidForAG("test~!@#^*()_-|}{[]\":;?><./pass"))

		// -[] have to be escaped in the regex
		assert.Equal(t, true, isStringValidForAG("pass-word"))
		assert.Equal(t, true, isStringValidForAG("pass[word"))
		assert.Equal(t, true, isStringValidForAG("pass]word"))
		assert.Equal(t, true, isStringValidForAG("pass$word"))

		// apostrophe
		assert.Equal(t, false, isStringValidForAG("pass'word"))
		// backtick
		assert.Equal(t, false, isStringValidForAG("pass`word"))
		// comma
		assert.Equal(t, false, isStringValidForAG("pass,word"))
		// ampersand
		assert.Equal(t, false, isStringValidForAG("pass&word"))
		// equals sign
		assert.Equal(t, false, isStringValidForAG("pass=word"))
		// plus
		assert.Equal(t, false, isStringValidForAG("pass+word"))

		// UTF-8 single character - U+1F600 grinning face
		assert.Equal(t, false, isStringValidForAG("\xF0\x9F\x98\x80"))
	})
}
