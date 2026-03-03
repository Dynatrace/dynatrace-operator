package activegate

import (
	"errors"
	"testing"

	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetAuthToken(t *testing.T) {
	ctx := t.Context()

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)

		request.EXPECT().WithJSONBody(mock.Anything).Return(request).Once()
		request.EXPECT().Execute(new(AuthTokenInfo)).Run(func(obj any) {
			target := obj.(*AuthTokenInfo)
			target.TokenID = "id"
			target.Token = "token"
		}).Return(nil).Once()

		apiClient.EXPECT().POST(ctx, authTokenPath).Return(request).Once()

		client := NewClient(apiClient)
		info, err := client.GetAuthToken(ctx, "dynakube")

		require.NoError(t, err)
		assert.Equal(t, "id", info.TokenID)
		assert.Equal(t, "token", info.Token)
	})

	t.Run("error", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)

		request.EXPECT().WithJSONBody(mock.Anything).Return(request).Once()
		request.EXPECT().Execute(new(AuthTokenInfo)).Return(errors.New("api error")).Once()

		apiClient.EXPECT().POST(ctx, authTokenPath).Return(request).Once()

		client := NewClient(apiClient)
		info, err := client.GetAuthToken(ctx, "dynakube")

		require.Error(t, err)
		assert.Nil(t, info)
	})
}

func TestGetConnectionInfo(t *testing.T) {
	ctx := t.Context()

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)

		request.EXPECT().WithPaasToken().Return(request).Once()
		request.EXPECT().Execute(new(connectionInfoJSONResponse)).Run(func(obj any) {
			target := obj.(*connectionInfoJSONResponse)
			target.TenantUUID = "uuid"
			target.TenantToken = "token"
			target.CommunicationEndpoints = "endpoints"
		}).Return(nil).Once()

		apiClient.EXPECT().GET(ctx, connectionInfoPath).Return(request).Once()

		client := NewClient(apiClient)
		info, err := client.GetConnectionInfo(ctx)

		require.NoError(t, err)
		assert.Equal(t, "uuid", info.TenantUUID)
		assert.Equal(t, "token", info.TenantToken)
		assert.Equal(t, "endpoints", info.Endpoints)
	})

	t.Run("error", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)

		request.EXPECT().WithPaasToken().Return(request).Once()
		request.EXPECT().Execute(new(connectionInfoJSONResponse)).Return(errors.New("api error")).Once()

		apiClient.EXPECT().GET(ctx, connectionInfoPath).Return(request).Once()

		client := NewClient(apiClient)
		info, err := client.GetConnectionInfo(ctx)

		require.Error(t, err)
		assert.Equal(t, ConnectionInfo{}, info)
	})

	t.Run("no endpoints", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)

		request.EXPECT().WithPaasToken().Return(request).Once()
		request.EXPECT().Execute(new(connectionInfoJSONResponse)).Run(func(obj any) {
			target := obj.(*connectionInfoJSONResponse)
			target.TenantUUID = "uuid"
			target.TenantToken = "token"
		}).Return(nil).Once()

		apiClient.EXPECT().GET(ctx, connectionInfoPath).Return(request).Once()

		client := NewClient(apiClient)
		info, err := client.GetConnectionInfo(ctx)

		require.NoError(t, err)
		assert.Equal(t, "uuid", info.TenantUUID)
		assert.Empty(t, info.Endpoints)
	})
}
