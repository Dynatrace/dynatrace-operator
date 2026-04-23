package dynatraceclient

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
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
	ctx := t.Context()

	t.Run("BuildDynatraceClient works with minimal setup", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testEndpoint,
			}}
		fakeClient := fake.NewClient(dk)
		dynatraceClientBuilder := builder{
			apiReader: fakeClient,
			tokens: map[string]*token.Token{
				token.APIKey:  {Value: testValue},
				token.PaaSKey: {Value: testValueAlternative},
			},
			dk: *dk,
		}
		dtClient, err := dynatraceClientBuilder.Build(ctx)

		require.NoError(t, err)
		assert.NotNil(t, dtClient)
	})
	t.Run("BuildDynatraceClient handles nil instance", func(t *testing.T) {
		dtClient, err := builder{}.Build(ctx)
		assert.Nil(t, dtClient)
		require.Error(t, err)
	})
	t.Run("BuildDynatraceClient handles invalid token secret", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testEndpoint,
			}}
		fakeClient := fake.NewClient(dk)
		dynatraceClientBuilder := builder{
			apiReader: fakeClient,
			tokens: map[string]*token.Token{
				// Simulate missing values
				token.APIKey:  {Value: ""},
				token.PaaSKey: {Value: ""},
			},
			dk: *dk,
		}

		dtClient, err := dynatraceClientBuilder.Build(ctx)

		assert.Nil(t, dtClient)
		require.Error(t, err)

		dynatraceClientBuilder = builder{
			apiReader: fakeClient,
			dk:        *dk,
		}
		dtClient, err = dynatraceClientBuilder.Build(ctx)

		assert.Nil(t, dtClient)
		require.Error(t, err)
	})
	t.Run("BuildDynatraceClient handles missing proxy secret", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testEndpoint,
				Proxy: &value.Source{
					ValueFrom: testKey,
				}}}
		fakeClient := fake.NewClient(dk)
		dynatraceClientBuilder := builder{
			apiReader: fakeClient,
			tokens: map[string]*token.Token{
				token.APIKey:  {Value: testValue},
				token.PaaSKey: {Value: testValueAlternative},
			},
			dk: *dk,
		}
		dtClient, err := dynatraceClientBuilder.Build(ctx)

		require.Error(t, err)
		assert.Nil(t, dtClient)
	})
	t.Run("BuildDynatraceClient handles missing trusted certificate config map", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:     testEndpoint,
				TrustedCAs: testKey,
			}}

		fakeClient := fake.NewClient(dk)
		dtf := builder{
			apiReader: fakeClient,
			tokens: map[string]*token.Token{
				token.APIKey:  {Value: testValue},
				token.PaaSKey: {Value: testValueAlternative},
			},
			dk: *dk,
		}
		dtClient, err := dtf.Build(ctx)

		require.Error(t, err)
		assert.Nil(t, dtClient)
	})
}
