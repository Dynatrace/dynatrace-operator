package edgeconnect

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const edgeConnectID = "test-id"

var edgeConnectRequest = NewRequest("InternalServices", []string{"*.internal.org"}, []edgeconnect.HostMapping{}, "dt0s02.AIOUP56P")

func TestCreateEdgeConnect(t *testing.T) {
	t.Run("create edge connect", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithJSONBody(edgeConnectRequest).Return(request).Once()
		request.EXPECT().Execute(new(CreateResponse)).Run(func(obj any) {
			target := obj.(*CreateResponse)
			target.Name = edgeConnectRequest.Name
		}).Return(nil).Once()
		apiClient.EXPECT().POST(mock.Anything, "/platform/app-engine/edge-connect/v1/edge-connects").Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		got, err := mockClient.CreateEdgeConnect(t.Context(), edgeConnectRequest)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, edgeConnectRequest.Name, got.Name)
	})

	t.Run("fail to create edge connect", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithJSONBody(edgeConnectRequest).Return(request).Once()
		request.EXPECT().Execute(new(CreateResponse)).Return(errTest).Once()
		apiClient.EXPECT().POST(mock.Anything, "/platform/app-engine/edge-connect/v1/edge-connects").Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		got, err := mockClient.CreateEdgeConnect(t.Context(), edgeConnectRequest)
		require.ErrorIs(t, err, errTest)
		require.Equal(t, CreateResponse{}, got)
	})
}

func TestUpdateEdgeConnect(t *testing.T) {
	t.Run("update edge connect", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithJSONBody(edgeConnectRequest).Return(request).Once()
		request.EXPECT().Execute(nil).Return(nil).Once()
		apiClient.EXPECT().PUT(mock.Anything, fmt.Sprintf("/platform/app-engine/edge-connect/v1/edge-connects/%s", edgeConnectID)).Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.UpdateEdgeConnect(t.Context(), edgeConnectID, edgeConnectRequest)
		require.NoError(t, err)
	})

	t.Run("fail to update edge connect", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithJSONBody(edgeConnectRequest).Return(request).Once()
		request.EXPECT().Execute(nil).Return(errTest).Once()
		apiClient.EXPECT().PUT(mock.Anything, fmt.Sprintf("/platform/app-engine/edge-connect/v1/edge-connects/%s", edgeConnectID)).Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.UpdateEdgeConnect(t.Context(), edgeConnectID, edgeConnectRequest)
		require.ErrorIs(t, err, errTest)
	})

	t.Run("fail if no edge connect id", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.UpdateEdgeConnect(t.Context(), "", edgeConnectRequest)
		require.EqualError(t, err, "no EdgeConnect ID given")
	})
}

func TestGetEdgeConnects(t *testing.T) {
	const name = "test-name"

	ecQp := map[string]string{
		"add-fields": "name,managedByDynatraceOperator",
		"filter":     fmt.Sprintf("name='%s'", name),
	}

	t.Run("get edge connects", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithQueryParams(ecQp).Return(request).Once()
		request.EXPECT().Execute(new(ListResponse)).Run(func(obj any) {
			target := obj.(*ListResponse)
			target.TotalCount = 1
			target.EdgeConnects = []GetResponse{
				{Name: name},
			}
		}).Return(nil).Once()
		apiClient.EXPECT().GET(mock.Anything, "/platform/app-engine/edge-connect/v1/edge-connects").Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		got, err := mockClient.GetEdgeConnects(t.Context(), name)
		require.NoError(t, err)
		require.Equal(t, 1, got.TotalCount)
		require.Equal(t, name, got.EdgeConnects[0].Name)
		require.Len(t, got.EdgeConnects, 1)
	})

	t.Run("fail to get edge connects", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().WithQueryParams(ecQp).Return(request).Once()
		request.EXPECT().Execute(new(ListResponse)).Return(errTest).Once()
		apiClient.EXPECT().GET(mock.Anything, "/platform/app-engine/edge-connect/v1/edge-connects").Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		got, err := mockClient.GetEdgeConnects(t.Context(), name)
		require.ErrorIs(t, err, errTest)
		require.Equal(t, ListResponse{}, got)
	})
}

func TestDeleteEdgeConnect(t *testing.T) {
	t.Run("delete edge connect", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().Execute(nil).Return(nil).Once()
		apiClient.EXPECT().DELETE(mock.Anything, fmt.Sprintf("/platform/app-engine/edge-connect/v1/edge-connects/%s", edgeConnectID)).Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.DeleteEdgeConnect(t.Context(), edgeConnectID)
		require.NoError(t, err)
	})

	t.Run("fail to delete edge connect", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithOAuthToken().Return(request).Once()
		request.EXPECT().Execute(nil).Return(errTest).Once()
		apiClient.EXPECT().DELETE(mock.Anything, fmt.Sprintf("/platform/app-engine/edge-connect/v1/edge-connects/%s", edgeConnectID)).Return(request).Once()
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.DeleteEdgeConnect(t.Context(), edgeConnectID)
		require.ErrorIs(t, err, errTest)
	})

	t.Run("fail if no edge connect id", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		mockClient := NewClientFromAPIClient(apiClient)
		err := mockClient.DeleteEdgeConnect(t.Context(), "")
		require.EqualError(t, err, "no EdgeConnect ID given")
	})
}
