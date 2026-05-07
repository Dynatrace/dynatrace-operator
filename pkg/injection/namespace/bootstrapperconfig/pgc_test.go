package bootstrapperconfig

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
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
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "").
			Return(&oneagent.ProcessGroupConfig{
				ETag: "etag123",
				Data: payload,
			}, nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		assert.Equal(t, payload, pgc)
	})

	t.Run("success - response between 800 KiB and 980 KiB triggers warning", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		payload := bytes.Repeat([]byte("a"), 750*1024) // 750 KiB — above warn, below max
		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "").
			Return(&oneagent.ProcessGroupConfig{
				ETag: "etag123",
				Data: payload,
			}, nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		assert.Equal(t, payload, pgc)
	})

	t.Run("response over 880 KiB returns nil without error", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		payload := bytes.Repeat([]byte("a"), 890*1024) // 890 KiB — above max
		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "").
			Return(&oneagent.ProcessGroupConfig{
				ETag: "etag123",
				Data: payload,
			}, nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		assert.Nil(t, pgc)
	})

	t.Run("API error is propagated and condition is set", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		expectedErr := errors.New("API error")
		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "").
			Return(nil, expectedErr)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		result, err := sg.preparePGC(t.Context(), dk)

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("empty response returns nil without error", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "").
			Return(nil, nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		result, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}
