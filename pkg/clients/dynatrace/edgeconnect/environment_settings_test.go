package edgeconnect

import (
	"errors"
	"fmt"
	"testing"

	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testObjectID = "test-objectId"

var testEnvironmentSetting = EnvironmentSetting{
	ObjectID: &testObjectID,
	SchemaID: KubernetesConnectionSchemaID,
	Scope:    KubernetesConnectionScope,
	Value: EnvironmentSettingValue{
		Name:      "test-name",
		UID:       "test-uid",
		Namespace: "test-namespace",
		Token:     "test-token",
	},
}

var qp = map[string]string{
	"schemaIds": "app:dynatrace.kubernetes.connector:connection",
	"scopes":    "environment",
}

var errTest = errors.New("test-error")

func TestGetConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithQueryParams(qp).Return(request).Once()
		request.EXPECT().Execute(new(EnvironmentSettingsResponse)).Run(func(obj any) {
			target := obj.(*EnvironmentSettingsResponse)
			target.Items = []EnvironmentSetting{
				testEnvironmentSetting,
			}
		}).Return(nil).Once()
		apiClient.EXPECT().GET(mock.Anything, "/platform/classic/environment-api/v2/settings/objects").Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		got, err := mockClient.GetConnectionSettings(t.Context())
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Len(t, got, 1)
		require.Equal(t, testEnvironmentSetting, got[0])
	})

	t.Run("Server response NOK", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().Execute(new(EnvironmentSettingsResponse)).Return(errTest).Once()
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithQueryParams(qp).Return(request).Once()
		apiClient.EXPECT().GET(mock.Anything, "/platform/classic/environment-api/v2/settings/objects").Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		got, err := mockClient.GetConnectionSettings(t.Context())
		require.ErrorIs(t, err, errTest)
		require.Nil(t, got)
	})
}

func TestCreateConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithJSONBody([]EnvironmentSetting{testEnvironmentSetting}).Return(request).Once()
		request.EXPECT().Execute(nil).Return(nil).Once()
		apiClient.EXPECT().POST(mock.Anything, "/platform/classic/environment-api/v2/settings/objects").Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.CreateConnectionSetting(t.Context(), testEnvironmentSetting)
		require.NoError(t, err)
	})

	t.Run("Server response NOK", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithJSONBody([]EnvironmentSetting{testEnvironmentSetting}).Return(request).Once()
		request.EXPECT().Execute(nil).Return(errTest).Once()
		apiClient.EXPECT().POST(mock.Anything, "/platform/classic/environment-api/v2/settings/objects").Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.CreateConnectionSetting(t.Context(), testEnvironmentSetting)
		require.ErrorIs(t, err, errTest)
	})
}

func TestUpdateConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithJSONBody(testEnvironmentSetting).Return(request).Once()
		request.EXPECT().Execute(nil).Return(nil).Once()
		apiClient.EXPECT().PUT(mock.Anything, fmt.Sprintf("/platform/classic/environment-api/v2/settings/objects/%s", testObjectID)).Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.UpdateConnectionSetting(t.Context(), testEnvironmentSetting)
		require.NoError(t, err)
	})

	t.Run("Server response NOK", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithJSONBody(testEnvironmentSetting).Return(request).Once()
		request.EXPECT().Execute(nil).Return(errTest).Once()
		apiClient.EXPECT().PUT(mock.Anything, fmt.Sprintf("/platform/classic/environment-api/v2/settings/objects/%s", testObjectID)).Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.UpdateConnectionSetting(t.Context(), testEnvironmentSetting)
		require.ErrorIs(t, err, errTest)
	})
}

func TestDeleteConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().Execute(nil).Return(nil).Once()
		apiClient.EXPECT().DELETE(mock.Anything, fmt.Sprintf("/platform/classic/environment-api/v2/settings/objects/%s", testObjectID)).Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.DeleteConnectionSetting(t.Context(), *testEnvironmentSetting.ObjectID)
		require.NoError(t, err)
	})

	t.Run("Server response NOK", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().Execute(nil).Return(errTest).Once()
		apiClient.EXPECT().DELETE(mock.Anything, fmt.Sprintf("/platform/classic/environment-api/v2/settings/objects/%s", testObjectID)).Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.DeleteConnectionSetting(t.Context(), *testEnvironmentSetting.ObjectID)
		require.ErrorIs(t, err, errTest)
	})

	t.Run("No object id given", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.DeleteConnectionSetting(t.Context(), "")
		require.EqualError(t, err, "no connection setting object id given")
	})
}
