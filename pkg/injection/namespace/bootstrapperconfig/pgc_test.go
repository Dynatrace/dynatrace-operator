// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_SecretGenerator_preparePGC(t *testing.T) {
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
		require.NotNil(t, pgc)
		assert.Equal(t, payload, pgc.Data)
		assert.Equal(t, "etag123", pgc.ETag)
	})

	t.Run("success - response between 800 KiB and 900 KiB triggers warning", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		payload := bytes.Repeat([]byte("a"), 850*1024) // 850 KiB — above warn, below max
		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "").
			Return(&oneagent.ProcessGroupConfig{
				ETag: "etag123",
				Data: payload,
			}, nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		require.NotNil(t, pgc)
		assert.Equal(t, payload, pgc.Data)
	})

	t.Run("response over 900 KiB returns nil without error", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		payload := bytes.Repeat([]byte("a"), 920*1024) // 920 KiB — above max
		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "").
			Return(&oneagent.ProcessGroupConfig{
				ETag: "etag123",
				Data: payload,
			}, nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		require.NotNil(t, pgc)
		assert.Empty(t, pgc.Data)
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
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, pgc)

		c := meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionFalse, c.Status)
	})

	t.Run("empty response returns nil without error", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "").
			Return(nil, nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		require.NotNil(t, pgc)
		assert.Empty(t, pgc.Data)
	})

	t.Run("304 not modified uses cached data", func(t *testing.T) {
		dk := newDK()
		cachedData := []byte("cached-pgc-data")
		cachedETag := "etag-abc"

		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetSourceConfigSecretName(testDynakube),
				Namespace: testNamespace,
				Annotations: map[string]string{
					annotationPGCETag: cachedETag,
				},
			},
			Data: map[string][]byte{
				DeclarativeInputFileName: cachedData,
			},
		}

		clt := fake.NewClient(dk, sourceSecret)
		mockDTClient := oneagentclientmock.NewClient(t)

		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, cachedETag).
			Return(&oneagent.ProcessGroupConfig{ETag: cachedETag}, nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		require.NotNil(t, pgc)
		assert.Equal(t, cachedData, pgc.Data)
		assert.Equal(t, cachedETag, pgc.ETag)
	})

	t.Run("empty MEID skips without error", func(t *testing.T) {
		dk := newDK()
		dk.Status.KubernetesClusterMEID = ""

		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		require.NotNil(t, pgc)
		assert.Empty(t, pgc.Data)
	})

	t.Run("200 response ETag is returned", func(t *testing.T) {
		dk := newDK()
		clt := fake.NewClient(dk)
		mockDTClient := oneagentclientmock.NewClient(t)

		payload := []byte("pgc-data")
		responseETag := "new-etag-xyz"

		mockDTClient.EXPECT().
			GetProcessGroupingConfig(mock.Anything, testClusterMEID, "").
			Return(&oneagent.ProcessGroupConfig{
				ETag: responseETag,
				Data: payload,
			}, nil)

		sg := NewSecretGenerator(clt, clt, mockDTClient)
		pgc, err := sg.preparePGC(t.Context(), dk)

		require.NoError(t, err)
		require.NotNil(t, pgc)
		assert.Equal(t, responseETag, pgc.ETag)
	})
}
