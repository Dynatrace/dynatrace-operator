package dynakube

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName             = "test-name"
	testNamespace        = "test-namespace"
	testEndpoint         = "https://test-endpoint.com"
	testValue            = "test-value"
	testKey              = "test-key"
	testValueAlternative = "test-alternative-value"
)

func TestBuildDynatraceClient(t *testing.T) {
	t.Run(`BuildDynatraceClient works with minimal setup`, func(t *testing.T) {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dtclient.DynatraceApiToken:  []byte(testValue),
				dtclient.DynatracePaasToken: []byte(testValueAlternative),
			}}
		instance := &dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: testEndpoint,
			}}
		fakeClient := fake.NewClient(instance, &secret)
		dtc, err := BuildDynatraceClient(fakeClient, instance, &secret)

		assert.NoError(t, err)
		assert.NotNil(t, dtc)
	})
	t.Run(`BuildDynatraceClient handles nil instance`, func(t *testing.T) {
		dtc, err := BuildDynatraceClient(nil, nil, nil)
		assert.Nil(t, dtc)
		assert.Error(t, err)
	})
	t.Run(`BuildDynatraceClient handles invalid token secret`, func(t *testing.T) {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				//Simulate missing values
			}}
		instance := &dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: testEndpoint,
			}}
		fakeClient := fake.NewClient(instance, &secret)
		dtc, err := BuildDynatraceClient(fakeClient, instance, &secret)
		assert.Nil(t, dtc)
		assert.Error(t, err)

		dtc, err = BuildDynatraceClient(fakeClient, instance, nil)
		assert.Nil(t, dtc)
		assert.Error(t, err)
	})
	t.Run(`BuildDynatraceClient handles missing proxy secret`, func(t *testing.T) {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dtclient.DynatraceApiToken:  []byte(testValue),
				dtclient.DynatracePaasToken: []byte(testValueAlternative),
			}}
		instance := &dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: testEndpoint,
				Proxy: &dynatracev1alpha1.DynaKubeProxy{
					ValueFrom: testKey,
				}}}
		fakeClient := fake.NewClient(instance, &secret)
		dtc, err := BuildDynatraceClient(fakeClient, instance, &secret)

		assert.Error(t, err)
		assert.Nil(t, dtc)
	})
	t.Run(`BuildDynatraceClient handles missing trusted certificate config map`, func(t *testing.T) {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dtclient.DynatraceApiToken:  []byte(testValue),
				dtclient.DynatracePaasToken: []byte(testValueAlternative),
			}}
		instance := &dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:     testEndpoint,
				TrustedCAs: testKey,
			}}

		fakeClient := fake.NewClient(instance, &secret)
		dtc, err := BuildDynatraceClient(fakeClient, instance, &secret)

		assert.Error(t, err)
		assert.Nil(t, dtc)
	})
}

func TestOptions(t *testing.T) {
	t.Run(`Test append network zone`, func(t *testing.T) {
		options := newOptions()

		assert.NotNil(t, options)
		assert.Empty(t, options.Opts)

		options.appendNetworkZone(&dynatracev1alpha1.DynaKubeSpec{})

		assert.Empty(t, options.Opts)

		options.appendNetworkZone(&dynatracev1alpha1.DynaKubeSpec{
			NetworkZone: testValue,
		})

		assert.NotEmpty(t, options.Opts)
	})
	t.Run(`Test append cert check`, func(t *testing.T) {
		options := newOptions()

		assert.NotNil(t, options)
		assert.Empty(t, options.Opts)

		options.appendCertCheck(&dynatracev1alpha1.DynaKubeSpec{})

		assert.NotNil(t, options)
		// appendCertCheck uses default value of property to append to,
		// which is why Opts is not empty although no value is given
		assert.NotEmpty(t, options.Opts)

		options = newOptions()
		options.appendCertCheck(&dynatracev1alpha1.DynaKubeSpec{
			SkipCertCheck: true,
		})

		assert.NotNil(t, options)
		assert.NotEmpty(t, options.Opts)
	})
	t.Run(`Test append proxy settings`, func(t *testing.T) {
		options := newOptions()

		assert.NotNil(t, options)
		assert.Empty(t, options.Opts)

		err := options.appendProxySettings(nil, &dynatracev1alpha1.DynaKubeSpec{}, "")
		assert.NoError(t, err)
		assert.Empty(t, options.Opts)

		err = options.appendProxySettings(nil, &dynatracev1alpha1.DynaKubeSpec{
			Proxy: &dynatracev1alpha1.DynaKubeProxy{
				Value: testValue,
			}}, "")

		assert.NoError(t, err)
		assert.NotEmpty(t, options.Opts)

		fakeClient := fake.NewClient(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					Proxy: []byte(testValue),
				},
			})
		options = newOptions()
		err = options.appendProxySettings(fakeClient, &dynatracev1alpha1.DynaKubeSpec{
			Proxy: &dynatracev1alpha1.DynaKubeProxy{
				ValueFrom: testName,
			}}, testNamespace)

		assert.NoError(t, err)
		assert.NotEmpty(t, options.Opts)
	})
	t.Run(`AppendProxySettings handles missing or malformed secret`, func(t *testing.T) {
		fakeClient := fake.NewClient()
		options := newOptions()
		err := options.appendProxySettings(fakeClient, &dynatracev1alpha1.DynaKubeSpec{
			Proxy: &dynatracev1alpha1.DynaKubeProxy{
				ValueFrom: testName,
			}}, testNamespace)

		assert.Error(t, err)
		assert.Empty(t, options.Opts)

		fakeClient = fake.NewClient(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{},
			})
		options = newOptions()
		err = options.appendProxySettings(fakeClient, &dynatracev1alpha1.DynaKubeSpec{
			Proxy: &dynatracev1alpha1.DynaKubeProxy{
				ValueFrom: testName,
			}}, testNamespace)

		assert.Error(t, err)
		assert.Empty(t, options.Opts)
	})
	t.Run(`Test append trusted certificates`, func(t *testing.T) {
		options := newOptions()

		assert.NotNil(t, options)
		assert.Empty(t, options.Opts)

		err := options.appendTrustedCerts(nil, &dynatracev1alpha1.DynaKubeSpec{}, "")

		assert.NoError(t, err)
		assert.Empty(t, options.Opts)

		fakeClient := fake.NewClient(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					Certificates: testValue,
				}})
		err = options.appendTrustedCerts(fakeClient, &dynatracev1alpha1.DynaKubeSpec{
			TrustedCAs: testName,
		}, testNamespace)

		assert.NoError(t, err)
		assert.NotEmpty(t, options.Opts)
	})
	t.Run(`AppendTrustedCerts handles missing or malformed config map`, func(t *testing.T) {
		options := newOptions()

		assert.NotNil(t, options)
		assert.Empty(t, options.Opts)

		fakeClient := fake.NewClient()
		err := options.appendTrustedCerts(fakeClient, &dynatracev1alpha1.DynaKubeSpec{
			TrustedCAs: testName,
		}, testNamespace)

		assert.Error(t, err)
		assert.Empty(t, options.Opts)

		fakeClient = fake.NewClient(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string]string{}})
		err = options.appendTrustedCerts(fakeClient, &dynatracev1alpha1.DynaKubeSpec{
			TrustedCAs: testName,
		}, testNamespace)

		assert.Error(t, err)
		assert.Empty(t, options.Opts)
	})
}
