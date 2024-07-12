package dynatraceclient

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace        = "test-namespace"
	testEndpoint         = "https://test-endpoint.com"
	testValue            = "test-value"
	testKey              = "test-key"
	testValueAlternative = "test-alternative-value"
)

func TestBuildDynatraceClient(t *testing.T) {
	t.Run(`BuildDynatraceClient works with minimal setup`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testEndpoint,
			}}
		fakeClient := fake.NewClient(instance)
		dynatraceClientBuilder := builder{
			apiReader: fakeClient,
			tokens: map[string]*token.Token{
				dtclient.ApiToken:  {Value: testValue},
				dtclient.PaasToken: {Value: testValueAlternative},
			},
			dynakube: *instance,
		}
		dtc, err := dynatraceClientBuilder.Build()

		require.NoError(t, err)
		assert.NotNil(t, dtc)
	})
	t.Run(`BuildDynatraceClient handles nil instance`, func(t *testing.T) {
		dtc, err := builder{}.Build()
		assert.Nil(t, dtc)
		require.Error(t, err)
	})
	t.Run(`BuildDynatraceClient handles invalid token secret`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testEndpoint,
			}}
		fakeClient := fake.NewClient(instance)
		dynatraceClientBuilder := builder{
			apiReader: fakeClient,
			tokens: map[string]*token.Token{
				// Simulate missing values
				dtclient.ApiToken:  {Value: ""},
				dtclient.PaasToken: {Value: ""},
			},
			dynakube: *instance,
		}

		dtc, err := dynatraceClientBuilder.Build()

		assert.Nil(t, dtc)
		require.Error(t, err)

		dynatraceClientBuilder = builder{
			apiReader: fakeClient,
			dynakube:  *instance,
		}
		dtc, err = dynatraceClientBuilder.Build()

		assert.Nil(t, dtc)
		require.Error(t, err)
	})
	t.Run(`BuildDynatraceClient handles missing proxy secret`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testEndpoint,
				Proxy: &dynakube.DynaKubeProxy{
					ValueFrom: testKey,
				}}}
		fakeClient := fake.NewClient(instance)
		dynatraceClientBuilder := builder{
			apiReader: fakeClient,
			tokens: map[string]*token.Token{
				dtclient.ApiToken:  {Value: testValue},
				dtclient.PaasToken: {Value: testValueAlternative},
			},
			dynakube: *instance,
		}
		dtc, err := dynatraceClientBuilder.Build()

		require.Error(t, err)
		assert.Nil(t, dtc)
	})
	t.Run(`BuildDynatraceClient handles missing trusted certificate config map`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:     testEndpoint,
				TrustedCAs: testKey,
			}}

		fakeClient := fake.NewClient(instance)
		dtf := builder{
			apiReader: fakeClient,
			tokens: map[string]*token.Token{
				dtclient.ApiToken:  {Value: testValue},
				dtclient.PaasToken: {Value: testValueAlternative},
			},
			dynakube: *instance,
		}
		dtc, err := dtf.Build()

		require.Error(t, err)
		assert.Nil(t, dtc)
	})
}
