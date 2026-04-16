package edgeconnect

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var testObjectID = "test-objectId"

var testEnvironmentSetting = EnvironmentSetting{
	ObjectID: testObjectID,
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

func TestListEnvironmentSettings(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectGET(t, apiClient, request, settingsObjectsPath, "")
		request.EXPECT().WithQueryParams(qp).Return(request).Once()
		request.EXPECT().Execute(new(environmentSettingsResponse)).Run(func(obj any) {
			obj.(*environmentSettingsResponse).Items = []EnvironmentSetting{
				testEnvironmentSetting,
			}
		}).Return(nil).Once()
		got, err := mockClient.ListEnvironmentSettings(t.Context())
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Len(t, got, 1)
		require.Equal(t, testEnvironmentSetting, got[0])
	})

	t.Run("Server response NOK", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectGET(t, apiClient, request, settingsObjectsPath, "")
		request.EXPECT().WithQueryParams(qp).Return(request).Once()
		request.EXPECT().Execute(new(environmentSettingsResponse)).Return(errTest).Once()
		got, err := mockClient.ListEnvironmentSettings(t.Context())
		require.ErrorIs(t, err, errTest)
		require.Nil(t, got)
	})
}

func TestCreateEnvironmentSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectPOST(t, apiClient, request, settingsObjectsPath, []EnvironmentSetting{testEnvironmentSetting})
		request.EXPECT().Execute(nil).Return(nil).Once()
		err := mockClient.CreateEnvironmentSetting(t.Context(), testEnvironmentSetting)
		require.NoError(t, err)
	})

	t.Run("Server response NOK", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectPOST(t, apiClient, request, settingsObjectsPath, []EnvironmentSetting{testEnvironmentSetting})
		request.EXPECT().Execute(nil).Return(errTest).Once()
		err := mockClient.CreateEnvironmentSetting(t.Context(), testEnvironmentSetting)
		require.ErrorIs(t, err, errTest)
	})
}

func TestUpdateEnvironmentSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectPUT(t, apiClient, request, settingsObjectsPath, testObjectID, testEnvironmentSetting)
		request.EXPECT().Execute(nil).Return(nil).Once()
		err := mockClient.UpdateEnvironmentSetting(t.Context(), testEnvironmentSetting)
		require.NoError(t, err)
	})

	t.Run("Server response NOK", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectPUT(t, apiClient, request, settingsObjectsPath, testObjectID, testEnvironmentSetting)
		request.EXPECT().Execute(nil).Return(errTest).Once()
		err := mockClient.UpdateEnvironmentSetting(t.Context(), testEnvironmentSetting)
		require.ErrorIs(t, err, errTest)
	})

	t.Run("No object id given", func(t *testing.T) {
		_, _, mockClient := newTestSetup(t)
		err := mockClient.UpdateEnvironmentSetting(t.Context(), EnvironmentSetting{})
		require.ErrorIs(t, err, errNoEnvSettingObjectID)
	})

	t.Run("No object id given", func(t *testing.T) {
		_, _, mockClient := newTestSetup(t)
		err := mockClient.UpdateEnvironmentSetting(t.Context(), EnvironmentSetting{ObjectID: ""})
		require.ErrorIs(t, err, errNoEnvSettingObjectID)
	})
}

func TestDeleteEnvironmentSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectDELETE(t, apiClient, request, settingsObjectsPath, testObjectID)
		request.EXPECT().Execute(nil).Return(nil).Once()
		err := mockClient.DeleteEnvironmentSetting(t.Context(), testEnvironmentSetting.ObjectID)
		require.NoError(t, err)
	})

	t.Run("Server response NOK", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectDELETE(t, apiClient, request, settingsObjectsPath, testObjectID)
		request.EXPECT().Execute(nil).Return(errTest).Once()
		err := mockClient.DeleteEnvironmentSetting(t.Context(), testEnvironmentSetting.ObjectID)
		require.ErrorIs(t, err, errTest)
	})

	t.Run("No object id given", func(t *testing.T) {
		_, _, mockClient := newTestSetup(t)
		err := mockClient.DeleteEnvironmentSetting(t.Context(), "")
		require.ErrorIs(t, err, errNoEnvSettingObjectID)
	})
}
