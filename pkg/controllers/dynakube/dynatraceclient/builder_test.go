package dynatraceclient

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
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
				dynatrace.DynatraceApiToken:  {Value: testValue},
				dynatrace.DynatracePaasToken: {Value: testValueAlternative},
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
				dynatrace.DynatraceApiToken:  {Value: ""},
				dynatrace.DynatracePaasToken: {Value: ""},
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
				dynatrace.DynatraceApiToken:  {Value: testValue},
				dynatrace.DynatracePaasToken: {Value: testValueAlternative},
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
				dynatrace.DynatraceApiToken:  {Value: testValue},
				dynatrace.DynatracePaasToken: {Value: testValueAlternative},
			},
			dynakube: *instance,
		}
		dtc, err := dtf.Build()

		assert.Error(t, err)
		assert.Nil(t, dtc)
	})
}
