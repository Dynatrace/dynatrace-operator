package bootstrapperconfig

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	oneagentclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_SecretGenerator_prepareDeclarativeConfig(t *testing.T) {
	const (
		testDynakube       = "dk"
		testNamespace      = "ns"
		testKubeSystemUUID = "kube-system-uuid"
		testClusterMEID    = "KUBERNETES_CLUSTER-test"
	)

	newDK := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID:        testKubeSystemUUID,
				KubernetesClusterMEID: testClusterMEID,
			},
		}
	}

	t.Run("success - response within size limit", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		payload := bytes.Repeat([]byte("a"), 100*1024) // 100 KiB
		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "", mock.AnythingOfType("*bytes.Buffer")).
			Run(func(_ context.Context, _ string, _ string, writer io.Writer) {
				_, _ = writer.Write(payload)
			}).
			Return("", nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		result, err := sg.prepareDeclarativeConfig(t.Context(), dk)

		require.NoError(t, err)
		assert.Equal(t, payload, result)
	})

	t.Run("success - response between 800 KiB and 980 KiB triggers warning", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		payload := bytes.Repeat([]byte("a"), 850*1024) // 850 KiB — above warn, below max
		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "", mock.AnythingOfType("*bytes.Buffer")).
			Run(func(_ context.Context, _ string, _ string, writer io.Writer) {
				_, _ = writer.Write(payload)
			}).
			Return("", nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		result, err := sg.prepareDeclarativeConfig(t.Context(), dk)

		require.NoError(t, err)
		assert.Equal(t, payload, result)
	})

	t.Run("response over 980 KiB returns nil without error", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		payload := bytes.Repeat([]byte("a"), 990*1024) // 990 KiB — above max
		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "", mock.AnythingOfType("*bytes.Buffer")).
			Run(func(_ context.Context, _ string, _ string, writer io.Writer) {
				_, _ = writer.Write(payload)
			}).
			Return("", nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		result, err := sg.prepareDeclarativeConfig(t.Context(), dk)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("API error is propagated and condition is set", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		expectedErr := errors.New("API error")
		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "", mock.AnythingOfType("*bytes.Buffer")).
			Return("", expectedErr)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		result, err := sg.prepareDeclarativeConfig(t.Context(), dk)

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("empty response returns nil without error", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "", mock.AnythingOfType("*bytes.Buffer")).
			Return("", nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		result, err := sg.prepareDeclarativeConfig(t.Context(), dk)

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}
