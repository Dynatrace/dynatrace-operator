package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testProxySecret = "proxysecret"

	// invalidPlainTextProxyURL contains forbidden apostrophe character.
	invalidPlainTextProxyURL = "http://test:password'!\"#$()*-./:;<>?@[]^_{|}~@proxy-service.dynatrace:3128"

	// validEncodedProxyURL contains no forbidden characters "http://test:password!"#$()*-./:;<>?@[]^_{|}~@proxy-service.dynatrace:3128"
	validEncodedProxyURL = "http://test:password!%22%23%24()*-.%2F%3A%3B%3C%3E%3F%40%5B%5D%5E_%7B%7C%7D~@proxy-service.dynatrace:3128"

	// validEncodedProxyURLNoPassword contains empty password.
	validEncodedProxyURLNoPassword = "http://test@proxy-service.dynatrace:3128"
)

func TestInvalidActiveGateProxy(t *testing.T) {
	t.Run(`valid proxy url`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Proxy: &value.Source{
						Value:     validEncodedProxyURL,
						ValueFrom: "",
					},
				},
			})
	})

	t.Run(`valid proxy url, no password`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Proxy: &value.Source{
						Value:     validEncodedProxyURLNoPassword,
						ValueFrom: "",
					},
				},
			})
	})

	t.Run(`invalid proxy url`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorInvalidProxyURL},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Proxy: &value.Source{
						Value:     invalidPlainTextProxyURL,
						ValueFrom: "",
					},
				},
			})
	})

	t.Run(`valid proxy secret url`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Proxy: &value.Source{
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
					"proxy": []byte(validEncodedProxyURL),
				},
			})
	})

	t.Run(`missing proxy secret`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorMissingProxySecret},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Proxy: &value.Source{
						Value:     "",
						ValueFrom: testProxySecret,
					},
				},
			})
	})

	t.Run(`invalid format of proxy secret`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorMissingProxySecret},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Proxy: &value.Source{
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
					"invalid-name": []byte(validEncodedProxyURL),
				},
			})
	})

	t.Run(`invalid proxy secret url`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorInvalidProxyURL},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					Proxy: &value.Source{
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
					"proxy": []byte(invalidPlainTextProxyURL),
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
