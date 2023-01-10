package dynatraceclient

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testEndpoint,
			}}
		fakeClient := fake.NewClient(instance)
		dynatraceClientBuilder := builder{
			apiReader: fakeClient,
			tokens: map[string]token.Token{
				dtclient.DynatraceApiToken:  {Value: testValue},
				dtclient.DynatracePaasToken: {Value: testValueAlternative},
			},
			dynakube: *instance,
		}
		dtc, err := dynatraceClientBuilder.Build()

		assert.NoError(t, err)
		assert.NotNil(t, dtc)
	})
	t.Run(`BuildDynatraceClient handles nil instance`, func(t *testing.T) {
		dtc, err := builder{}.Build()
		assert.Nil(t, dtc)
		assert.Error(t, err)
	})
	t.Run(`BuildDynatraceClient handles invalid token secret`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testEndpoint,
			}}
		fakeClient := fake.NewClient(instance)
		dynatraceClientBuilder := builder{
			apiReader: fakeClient,
			tokens: map[string]token.Token{
				// Simulate missing values
				dtclient.DynatraceApiToken:  {Value: ""},
				dtclient.DynatracePaasToken: {Value: ""},
			},
			dynakube: *instance,
		}

		dtc, err := dynatraceClientBuilder.Build()

		assert.Nil(t, dtc)
		assert.Error(t, err)

		dynatraceClientBuilder = builder{
			apiReader: fakeClient,
			dynakube:  *instance,
		}
		dtc, err = dynatraceClientBuilder.Build()

		assert.Nil(t, dtc)
		assert.Error(t, err)
	})
	t.Run(`BuildDynatraceClient handles missing proxy secret`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testEndpoint,
				Proxy: &dynatracev1beta1.DynaKubeProxy{
					ValueFrom: testKey,
				}}}
		fakeClient := fake.NewClient(instance)
		dynatraceClientBuilder := builder{
			apiReader: fakeClient,
			tokens: map[string]token.Token{
				dtclient.DynatraceApiToken:  {Value: testValue},
				dtclient.DynatracePaasToken: {Value: testValueAlternative},
			},
			dynakube: *instance,
		}
		dtc, err := dynatraceClientBuilder.Build()

		assert.Error(t, err)
		assert.Nil(t, dtc)
	})
	t.Run(`BuildDynatraceClient handles missing trusted certificate config map`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL:     testEndpoint,
				TrustedCAs: testKey,
			}}

		fakeClient := fake.NewClient(instance)
		dtf := builder{
			apiReader: fakeClient,
			tokens: map[string]token.Token{
				dtclient.DynatraceApiToken:  {Value: testValue},
				dtclient.DynatracePaasToken: {Value: testValueAlternative},
			},
			dynakube: *instance,
		}
		dtc, err := dtf.Build()

		assert.Error(t, err)
		assert.Nil(t, dtc)
	})
}
