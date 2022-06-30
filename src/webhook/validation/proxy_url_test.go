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
	invalidPlainTextProxyUrl = "http://test:password'!\"#$()*-./:;<>?@[]^_{|}~@proxy-service.dynatrace:3128"

	// validEncodedProxyUrl contains no forbidden characters "http://test:password!"#$()*-./:;<>?@[]^_{|}~@proxy-service.dynatrace:3128"
	validEncodedProxyUrl = "http://test:password!%22%23%24()*-.%2F%3A%3B%3C%3E%3F%40%5B%5D%5E_%7B%7C%7D~@proxy-service.dynatrace:3128"

	// validEncodedProxyUrlNoPassword contains empty password
	validEncodedProxyUrlNoPassword = "http://test@proxy-service.dynatrace:3128"
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

	t.Run(`valid proxy url, no password`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     validEncodedProxyUrlNoPassword,
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
		assert.True(t, isStringValidForAG("password"))
		assert.True(t, isStringValidForAG("test~!@#^*()_-|}{[]\":;?><./pass"))

		// -[] have to be escaped in the regex
		assert.True(t, isStringValidForAG("pass-word"))
		assert.True(t, isStringValidForAG("pass[word"))
		assert.True(t, isStringValidForAG("pass]word"))
		assert.True(t, isStringValidForAG("pass$word"))

		// apostrophe
		assert.False(t, isStringValidForAG("pass'word"))
		// backtick
		assert.False(t, isStringValidForAG("pass`word"))
		// comma
		assert.False(t, isStringValidForAG("pass,word"))
		// ampersand
		assert.False(t, isStringValidForAG("pass&word"))
		// equals sign
		assert.False(t, isStringValidForAG("pass=word"))
		// plus sign
		assert.False(t, isStringValidForAG("pass+word"))
		// percent sign
		assert.False(t, isStringValidForAG("pass%word"))
		// backslash
		assert.False(t, isStringValidForAG("pass\\word"))

		// UTF-8 single character - U+1F600 grinning face
		assert.False(t, isStringValidForAG("\xF0\x9F\x98\x80"))
	})
}
