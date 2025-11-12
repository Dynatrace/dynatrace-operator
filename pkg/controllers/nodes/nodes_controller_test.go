package nodes

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/nodes/cache"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testNamespace = "dynatrace"
	testAPIToken  = "test-api-token"
)

func TestReconcile(t *testing.T) {
	t.Run("Create node and then delete it", func(t *testing.T) {
		ctx := t.Context()
		node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}

		fakeClient := fake.NewClient(
			node,
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: "oneagent1", Namespace: testNamespace},
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						Instances: map[string]oneagent.Instance{node.Name: {IPAddress: "1.2.3.4"}},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oneagent1",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					dtclient.APIToken: []byte(testAPIToken),
				},
			},
		)

		dtClient := createDTMockClient(t, "1.2.3.4", "HOST-42")
		defer mock.AssertExpectationsForObjects(t, dtClient)

		ctrl := createDefaultReconciler(fakeClient, dtClient)
		result, err := ctrl.Reconcile(ctx, createReconcileRequest("node1"))
		require.NoError(t, err)
		assert.NotNil(t, result)

		// delete node from kube api
		err = fakeClient.Delete(ctx, node)
		require.NoError(t, err)

		// run another request reconcile
		result, err = ctrl.Reconcile(ctx, createReconcileRequest("node1"))
		require.NoError(t, err)
		assert.NotNil(t, result)

		nodesCache, err := cache.New(ctx, fakeClient, testNamespace, nil)
		require.NoError(t, err)

		_, err = nodesCache.GetEntry("node1")
		require.Error(t, err)
	})
	t.Run("No error if v1 host entity api is not present on tenant ", func(t *testing.T) {
		ctx := t.Context()
		node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}

		fakeClient := fake.NewClient(
			node,
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: "oneagent1", Namespace: testNamespace},
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						Instances: map[string]oneagent.Instance{node.Name: {IPAddress: "1.2.3.4"}},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oneagent1",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					dtclient.APIToken: []byte(testAPIToken),
				},
			},
		)

		dtClient := dtclientmock.NewClient(t)
		dtClient.On("GetHostEntityIDForIP", mock.AnythingOfType("*context.cancelCtx"), mock.Anything).Return("", dtclient.V1HostEntityAPINotAvailableErr{APIURL: "test"})

		ctrl := createDefaultReconciler(fakeClient, dtClient)
		result, err := ctrl.Reconcile(ctx, createReconcileRequest("node1"))
		require.NoError(t, err)
		assert.NotNil(t, result)

		// delete node from kube api
		err = fakeClient.Delete(ctx, node)
		require.NoError(t, err)

		// run another request reconcile
		result, err = ctrl.Reconcile(ctx, createReconcileRequest("node1"))
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("No error if v1 events api is not present on tenant ", func(t *testing.T) {
		ctx := t.Context()
		node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}

		fakeClient := fake.NewClient(
			node,
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: "oneagent1", Namespace: testNamespace},
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						Instances: map[string]oneagent.Instance{node.Name: {IPAddress: "1.2.3.4"}},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oneagent1",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					dtclient.APIToken: []byte(testAPIToken),
				},
			},
		)

		dtClient := dtclientmock.NewClient(t)
		dtClient.On("GetHostEntityIDForIP", mock.AnythingOfType("*context.cancelCtx"), "1.2.3.4").Return("HOST-42", nil)
		dtClient.On("SendEvent", mock.AnythingOfType("*context.cancelCtx"), mock.MatchedBy(func(e *dtclient.EventData) bool {
			return e.EventType == "MARKED_FOR_TERMINATION"
		})).Return(dtclient.V1EventsAPINotAvailableErr{APIURL: "test"})

		ctrl := createDefaultReconciler(fakeClient, dtClient)
		result, err := ctrl.Reconcile(ctx, createReconcileRequest("node1"))
		require.NoError(t, err)
		assert.NotNil(t, result)

		// delete node from kube api
		err = fakeClient.Delete(ctx, node)
		require.NoError(t, err)

		// run another request reconcile
		result, err = ctrl.Reconcile(ctx, createReconcileRequest("node1"))
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Create two nodes and then delete one", func(t *testing.T) {
		ctx := t.Context()
		node1 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
		fakeClient := createDefaultFakeClient()

		dtClient := createDTMockClient(t, "1.2.3.4", "HOST-42")
		defer mock.AssertExpectationsForObjects(t, dtClient)

		ctrl := createDefaultReconciler(fakeClient, dtClient)
		reconcileAllNodes(t, ctrl, fakeClient)

		// delete node from kube api
		err := fakeClient.Delete(ctx, node1)
		require.NoError(t, err)

		// run another request reconcile
		result, err := ctrl.Reconcile(ctx, createReconcileRequest("node1"))
		require.NoError(t, err)
		assert.NotNil(t, result)

		nodesCache, err := cache.New(ctx, fakeClient, testNamespace, nil)
		require.NoError(t, err)

		_, err = nodesCache.GetEntry("node1")
		require.Error(t, err)
		_, err = nodesCache.GetEntry("node2")
		require.NoError(t, err)
	})

	t.Run("Node has taint", func(t *testing.T) {
		ctx := t.Context()
		fakeClient := createDefaultFakeClient()
		dtClient := createDTMockClient(t, "1.2.3.4", "HOST-42")
		ctrl := createDefaultReconciler(fakeClient, dtClient)

		// Get node 1
		node1 := &corev1.Node{}
		err := fakeClient.Get(ctx, client.ObjectKey{Name: "node1"}, node1)
		require.NoError(t, err)

		reconcileAllNodes(t, ctrl, fakeClient)
		// Add taint that makes it unschedulable
		node1.Spec.Taints = []corev1.Taint{
			{Key: "ToBeDeletedByClusterAutoscaler"},
		}
		err = fakeClient.Update(ctx, node1)
		require.NoError(t, err)

		result, err := ctrl.Reconcile(ctx, createReconcileRequest("node1"))
		assert.NotNil(t, result)
		require.NoError(t, err)

		// Get node from cache
		c, err := ctrl.getCache(ctx)
		require.NoError(t, err)
		assert.NotNil(t, c)

		node, err := c.GetEntry("node1")
		require.NoError(t, err)
		assert.NotNil(t, node)

		// Check if LastMarkedForTermination Timestamp is set to current time
		// Added one minute buffer to account for operation times
		now := time.Now().UTC()
		assert.True(t, node.LastMarkedForTermination.Add(time.Minute).After(now))
	})

	t.Run("Server error when removing node", func(t *testing.T) {
		ctx := t.Context()
		fakeClient := createDefaultFakeClient()

		dtClient := dtclientmock.NewClient(t)
		dtClient.On("GetHostEntityIDForIP", mock.AnythingOfType("*context.cancelCtx"), mock.Anything).Return("", cache.ErrEntryNotFound)

		ctrl := createDefaultReconciler(fakeClient, dtClient)

		reconcileAllNodes(t, ctrl, fakeClient)

		// Get node from cache
		c, err := ctrl.getCache(ctx)
		require.NoError(t, err)

		require.ErrorIs(t, ctrl.reconcileNodeDeletion(ctx, c, "node1"), cache.ErrEntryNotFound)
	})

	t.Run("Remove host from cache even if server error: host not found", func(t *testing.T) {
		ctx := t.Context()
		fakeClient := createDefaultFakeClient()

		dtClient := dtclientmock.NewClient(t)
		dtClient.On("GetHostEntityIDForIP", mock.AnythingOfType("*context.cancelCtx"), mock.Anything).Return("", dtclient.HostEntityNotFoundErr{IP: "1.2.3.4"})

		ctrl := createDefaultReconciler(fakeClient, dtClient)

		reconcileAllNodes(t, ctrl, fakeClient)

		// Get node from cache
		c, err := ctrl.getCache(ctx)
		require.NoError(t, err)
		assert.NotNil(t, c)

		require.NoError(t, ctrl.reconcileNodeDeletion(ctx, c, "node1"))

		// should return not found for key inside configmap
		_, err = c.GetEntry("node1")
		require.ErrorIs(t, err, cache.ErrEntryNotFound)
	})

	t.Run("Handle outdated cache", func(t *testing.T) {
		ctx := t.Context()
		fakeClient := createDefaultFakeClient()

		dtClient := createDTMockClient(t, "1.2.3.4", "HOST-42")
		defer mock.AssertExpectationsForObjects(t, dtClient)

		ctrl := createDefaultReconciler(fakeClient, dtClient)
		// by doing this step we warm up cache by adding node1 and node2
		reconcileAllNodes(t, ctrl, fakeClient)

		// Emulate error by explicitly removing node1 from cache
		nodesCache, err := cache.New(ctx, fakeClient, testNamespace, nil)
		require.NoError(t, err)

		// delete node from kube api
		node1 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
		err = fakeClient.Delete(ctx, node1)
		require.NoError(t, err)

		require.NoError(t, ctrl.pruneCache(ctx, nodesCache))
	})
}

func createReconcileRequest(nodeName string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: nodeName},
	}
}

type mockDynatraceClientBuilder struct {
	dynatraceClient dtclient.Client
}

func (builder mockDynatraceClientBuilder) SetContext(context.Context) dynatraceclient.Builder {
	return builder
}

func (builder mockDynatraceClientBuilder) SetDynakube(dynakube.DynaKube) dynatraceclient.Builder {
	return builder
}

func (builder mockDynatraceClientBuilder) SetTokens(token.Tokens) dynatraceclient.Builder {
	return builder
}

func (builder mockDynatraceClientBuilder) LastAPIProbeTimestamp() *metav1.Time {
	return nil
}

func (builder mockDynatraceClientBuilder) Build(ctx context.Context) (dtclient.Client, error) {
	return builder.dynatraceClient, nil
}

func (builder mockDynatraceClientBuilder) BuildWithTokenVerification(*dynakube.DynaKubeStatus) (dtclient.Client, error) {
	return builder.dynatraceClient, nil
}

func createDefaultReconciler(fakeClient client.Client, dtClient *dtclientmock.Client) *Controller {
	return &Controller{
		client:    fakeClient,
		apiReader: fakeClient,
		dynatraceClientBuilder: &mockDynatraceClientBuilder{
			dynatraceClient: dtClient,
		},
		podNamespace: testNamespace,
		runLocal:     true,
		timeProvider: timeprovider.New().Freeze(),
	}
}

func createDTMockClient(t *testing.T, ip, host string) *dtclientmock.Client {
	dtClient := dtclientmock.NewClient(t)
	dtClient.On("GetHostEntityIDForIP", mock.AnythingOfType("*context.cancelCtx"), ip).Return(host, nil)
	dtClient.On("SendEvent", mock.AnythingOfType("*context.cancelCtx"), mock.MatchedBy(func(e *dtclient.EventData) bool {
		return e.EventType == "MARKED_FOR_TERMINATION"
	})).Return(nil)

	return dtClient
}

func reconcileAllNodes(t *testing.T, ctrl *Controller, fakeClient client.Client) {
	ctx := context.Background()

	var nodeList corev1.NodeList
	err := fakeClient.List(ctx, &nodeList)

	require.NoError(t, err)

	for _, clusterNode := range nodeList.Items {
		result, err := ctrl.Reconcile(ctx, createReconcileRequest(clusterNode.Name))
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func createDefaultFakeClient() client.Client {
	return fake.NewClient(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}},
		&dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent1", Namespace: testNamespace},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					Instances: map[string]oneagent.Instance{"node1": {IPAddress: "1.2.3.4"}},
				},
			},
		},
		&dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent2", Namespace: testNamespace},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					Instances: map[string]oneagent.Instance{"node2": {IPAddress: "5.6.7.8"}},
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oneagent1",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dtclient.APIToken: []byte(testAPIToken),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oneagent2",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dtclient.APIToken: []byte(testAPIToken),
			},
		})
}
