package edgeconnect

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/require"
)

const edgeConnectID = "test-id"

func TestGetEdgeConnect(t *testing.T) {
	t.Run("get EdgeConnect", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectGET(t, apiClient, request, edgeConnectsPath, edgeConnectID)
		request.EXPECT().Execute(new(APIResponse)).Run(func(obj any) {
			obj.(*APIResponse).Name = "test-name"
		}).Return(nil).Once()
		got, err := mockClient.GetEdgeConnect(t.Context(), edgeConnectID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "test-name", got.Name)
	})

	t.Run("fail to get EdgeConnect", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectGET(t, apiClient, request, edgeConnectsPath, edgeConnectID)
		request.EXPECT().Execute(new(APIResponse)).Return(errTest).Once()
		got, err := mockClient.GetEdgeConnect(t.Context(), edgeConnectID)
		require.ErrorIs(t, err, errTest)
		require.Equal(t, APIResponse{}, got)
	})

	t.Run("fail if no EdgeConnect ID", func(t *testing.T) {
		_, _, mockClient := newTestSetup(t)
		got, err := mockClient.GetEdgeConnect(t.Context(), "")
		require.ErrorIs(t, err, errNoEdgeConnectID)
		require.Equal(t, APIResponse{}, got)
	})
}

func TestCreateEdgeConnect(t *testing.T) {
	edgeConnectCreateRequest := NewCreateRequest("InternalServices", []string{"*.internal.org"}, []edgeconnect.HostMapping{})

	t.Run("create EdgeConnect", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectPOST(t, apiClient, request, edgeConnectsPath, edgeConnectCreateRequest)
		request.EXPECT().Execute(new(APIResponse)).Run(func(obj any) {
			obj.(*APIResponse).Name = edgeConnectCreateRequest.Name
		}).Return(nil).Once()
		got, err := mockClient.CreateEdgeConnect(t.Context(), edgeConnectCreateRequest)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, edgeConnectCreateRequest.Name, got.Name)
	})

	t.Run("fail to create EdgeConnect", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectPOST(t, apiClient, request, edgeConnectsPath, edgeConnectCreateRequest)
		request.EXPECT().Execute(new(APIResponse)).Return(errTest).Once()
		got, err := mockClient.CreateEdgeConnect(t.Context(), edgeConnectCreateRequest)
		require.ErrorIs(t, err, errTest)
		require.Equal(t, APIResponse{}, got)
	})
}

func TestUpdateEdgeConnect(t *testing.T) {
	edgeConnectUpdateRequest := NewUpdateRequest("InternalServices", []string{"*.internal.org"}, []edgeconnect.HostMapping{}, "dt0s02.AIOUP56P")

	t.Run("update EdgeConnect", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectPUT(t, apiClient, request, edgeConnectsPath, edgeConnectID, edgeConnectUpdateRequest)
		request.EXPECT().Execute(nil).Return(nil).Once()
		err := mockClient.UpdateEdgeConnect(t.Context(), edgeConnectID, edgeConnectUpdateRequest)
		require.NoError(t, err)
	})

	t.Run("fail to update EdgeConnect", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectPUT(t, apiClient, request, edgeConnectsPath, edgeConnectID, edgeConnectUpdateRequest)
		request.EXPECT().Execute(nil).Return(errTest).Once()
		err := mockClient.UpdateEdgeConnect(t.Context(), edgeConnectID, edgeConnectUpdateRequest)
		require.ErrorIs(t, err, errTest)
	})

	t.Run("fail if no EdgeConnect ID", func(t *testing.T) {
		_, _, mockClient := newTestSetup(t)
		err := mockClient.UpdateEdgeConnect(t.Context(), "", edgeConnectUpdateRequest)
		require.ErrorIs(t, err, errNoEdgeConnectID)
	})
}

func TestListEdgeConnects(t *testing.T) {
	const name = "test-name"

	ecQp := map[string]string{
		"add-fields": "name,managedByDynatraceOperator",
		"filter":     fmt.Sprintf("name='%s'", name),
	}

	t.Run("get EdgeConnects", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectGET(t, apiClient, request, edgeConnectsPath, "")
		request.EXPECT().WithQueryParams(ecQp).Return(request).Once()
		request.EXPECT().Execute(new(listResponse)).Run(func(obj any) {
			obj.(*listResponse).EdgeConnects = []APIResponse{
				{Name: name},
			}
		}).Return(nil).Once()
		got, err := mockClient.ListEdgeConnects(t.Context(), name)
		require.NoError(t, err)
		require.Equal(t, name, got[0].Name)
		require.Len(t, got, 1)
	})

	t.Run("fail to get EdgeConnects", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectGET(t, apiClient, request, edgeConnectsPath, "")
		request.EXPECT().WithQueryParams(ecQp).Return(request).Once()
		request.EXPECT().Execute(new(listResponse)).Return(errTest).Once()
		got, err := mockClient.ListEdgeConnects(t.Context(), name)
		require.ErrorIs(t, err, errTest)
		require.Equal(t, []APIResponse{}, got)
	})
}

func TestDeleteEdgeConnect(t *testing.T) {
	t.Run("delete EdgeConnect", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectDELETE(t, apiClient, request, edgeConnectsPath, edgeConnectID)
		request.EXPECT().Execute(nil).Return(nil).Once()
		err := mockClient.DeleteEdgeConnect(t.Context(), edgeConnectID)
		require.NoError(t, err)
	})

	t.Run("fail to delete EdgeConnect", func(t *testing.T) {
		apiClient, request, mockClient := newTestSetup(t)
		expectDELETE(t, apiClient, request, edgeConnectsPath, edgeConnectID)
		request.EXPECT().Execute(nil).Return(errTest).Once()
		err := mockClient.DeleteEdgeConnect(t.Context(), edgeConnectID)
		require.ErrorIs(t, err, errTest)
	})

	t.Run("fail if no EdgeConnect ID", func(t *testing.T) {
		_, _, mockClient := newTestSetup(t)
		err := mockClient.DeleteEdgeConnect(t.Context(), "")
		require.ErrorIs(t, err, errNoEdgeConnectID)
	})
}

func newTestSetup(t *testing.T) (*coremock.Client, *coremock.Request, *client) {
	t.Helper()
	apiClient := coremock.NewClient(t)
	request := coremock.NewRequest(t)

	return apiClient, request, NewClient(apiClient)
}

func expectGET(t *testing.T, apiClient *coremock.Client, request *coremock.Request, path, id string) {
	t.Helper()
	request.EXPECT().WithoutToken().Return(request).Once()
	if id != "" {
		request.EXPECT().WithPath([]string{id}).Return(request).Once()
	}
	apiClient.EXPECT().GET(t.Context(), path).Return(request).Once()
}

func expectPOST(t *testing.T, apiClient *coremock.Client, request *coremock.Request, path string, body any) {
	t.Helper()
	request.EXPECT().WithoutToken().Return(request).Once()
	request.EXPECT().WithJSONBody(body).Return(request).Once()
	apiClient.EXPECT().POST(t.Context(), path).Return(request).Once()
}

func expectPUT(t *testing.T, apiClient *coremock.Client, request *coremock.Request, path, id string, body any) {
	t.Helper()
	request.EXPECT().WithoutToken().Return(request).Once()
	request.EXPECT().WithJSONBody(body).Return(request).Once()
	request.EXPECT().WithPath([]string{id}).Return(request).Once()
	apiClient.EXPECT().PUT(t.Context(), path).Return(request).Once()
}

// expectDELETE sets up WithoutToken and DELETE mock expectations for the given path.
func expectDELETE(t *testing.T, apiClient *coremock.Client, request *coremock.Request, path, id string) {
	t.Helper()
	request.EXPECT().WithoutToken().Return(request).Once()
	request.EXPECT().WithPath([]string{id}).Return(request).Once()
	apiClient.EXPECT().DELETE(t.Context(), path).Return(request).Once()
}
