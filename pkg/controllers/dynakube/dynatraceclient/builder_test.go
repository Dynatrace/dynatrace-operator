package dynatraceclient

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
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
	ctx := context.Background()

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
				dtclient.APIToken:  {Value: testValue},
				dtclient.PaasToken: {Value: testValueAlternative},
			},
			dk: *dk,
		}
		dtc, err := dynatraceClientBuilder.Build(ctx)

		require.NoError(t, err)
		assert.NotNil(t, dtc)
	})
	t.Run("BuildDynatraceClient handles nil instance", func(t *testing.T) {
		dtc, err := builder{}.Build(ctx)
		assert.Nil(t, dtc)
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
				dtclient.APIToken:  {Value: ""},
				dtclient.PaasToken: {Value: ""},
			},
			dk: *dk,
		}

		dtc, err := dynatraceClientBuilder.Build(ctx)

		assert.Nil(t, dtc)
		require.Error(t, err)

		dynatraceClientBuilder = builder{
			apiReader: fakeClient,
			dk:        *dk,
		}
		dtc, err = dynatraceClientBuilder.Build(ctx)

		assert.Nil(t, dtc)
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
				dtclient.APIToken:  {Value: testValue},
				dtclient.PaasToken: {Value: testValueAlternative},
			},
			dk: *dk,
		}
		dtc, err := dynatraceClientBuilder.Build(ctx)

		require.Error(t, err)
		assert.Nil(t, dtc)
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
				dtclient.APIToken:  {Value: testValue},
				dtclient.PaasToken: {Value: testValueAlternative},
			},
			dk: *dk,
		}
		dtc, err := dtf.Build(ctx)

		require.Error(t, err)
		assert.Nil(t, dtc)
	})
}
