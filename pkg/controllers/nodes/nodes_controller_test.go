package nodes

import (
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/hostevent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/nodes/cache"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	hostclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/hostevent"
	dtbuildermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
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
					token.APIKey: []byte(testAPIToken),
				},
			},
		)

		dtClient := createDTMockClient(t, "1.2.3.4", "HOST-42")

		ctrl := createDefaultReconciler(t, fakeClient, dtClient)
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
					token.APIKey: []byte(testAPIToken),
				},
			},
		)

		hostClient := hostclientmock.NewAPIClient(t)
		hostClient.EXPECT().GetEntityIDForIP(t.Context(), "1.2.3.4").Return("", &core.HTTPError{StatusCode: 404})

		ctrl := createDefaultReconciler(t, fakeClient, &dynatrace.Client{HostEvent: hostClient})
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
					token.APIKey: []byte(testAPIToken),
				},
			},
		)

		hostClient := hostclientmock.NewAPIClient(t)
		hostClient.EXPECT().GetEntityIDForIP(t.Context(), "1.2.3.4").Return("HOST-42", nil).Once()
		hostClient.EXPECT().SendEvent(t.Context(), mock.MatchedBy(func(e hostevent.Event) bool {
			return e.EventType == hostevent.MarkedForTerminationEvent
		})).Return(&core.HTTPError{StatusCode: 404}).Once()

		ctrl := createDefaultReconciler(t, fakeClient, &dynatrace.Client{HostEvent: hostClient})
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
		defer mock.AssertExpectationsForObjects(t, dtClient.HostEvent)

		ctrl := createDefaultReconciler(t, fakeClient, dtClient)
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
		ctrl := createDefaultReconciler(t, fakeClient, dtClient)

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

		hostClient := hostclientmock.NewAPIClient(t)
		ctrl := createDefaultReconciler(t, fakeClient, &dynatrace.Client{HostEvent: hostClient})

		reconcileAllNodes(t, ctrl, fakeClient)

		// Get node from cache
		c, err := ctrl.getCache(ctx)
		require.NoError(t, err)

		// Only EXPECT to call something on deletion
		expectErr := errors.New("error")
		hostClient.EXPECT().GetEntityIDForIP(t.Context(), "1.2.3.4").Return("", expectErr)

		require.ErrorIs(t, ctrl.reconcileNodeDeletion(ctx, c, "node1"), expectErr)
	})

	t.Run("Remove host from cache even if server error: host not found", func(t *testing.T) {
		ctx := t.Context()
		fakeClient := createDefaultFakeClient()

		hostClient := hostclientmock.NewAPIClient(t)
		hostClient.EXPECT().GetEntityIDForIP(t.Context(), "1.2.3.4").Return("", hostevent.EntityNotFoundError{IP: "1.2.3.4"})

		ctrl := createDefaultReconciler(t, fakeClient, &dynatrace.Client{HostEvent: hostClient})

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
		defer mock.AssertExpectationsForObjects(t, dtClient.HostEvent)

		ctrl := createDefaultReconciler(t, fakeClient, dtClient)
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
		NamespacedName: types.NamespacedName{Name: nodeName},
	}
}

func createDefaultReconciler(t *testing.T, fakeClient client.Client, dtClient *dynatrace.Client) *Controller {
	mockDtcBuilder := dtbuildermock.NewBuilder(t)
	mockDtcBuilder.EXPECT().SetDynakube(mock.MatchedBy(func(dynakube.DynaKube) bool { return true })).Return(mockDtcBuilder)
	mockDtcBuilder.EXPECT().SetTokens(mock.MatchedBy(func(token.Tokens) bool { return true })).Return(mockDtcBuilder)
	mockDtcBuilder.EXPECT().Build(t.Context()).Return(dtClient, nil)

	return &Controller{
		client:                 fakeClient,
		apiReader:              fakeClient,
		dynatraceClientBuilder: mockDtcBuilder,
		podNamespace:           testNamespace,
		runLocal:               true,
		timeProvider:           timeprovider.New().Freeze(),
	}
}

func createDTMockClient(t *testing.T, ip, host string) *dynatrace.Client {
	hostClient := hostclientmock.NewAPIClient(t)
	hostClient.EXPECT().GetEntityIDForIP(t.Context(), ip).Return(host, nil)
	hostClient.EXPECT().SendEvent(t.Context(), mock.MatchedBy(func(e hostevent.Event) bool {
		return e.EventType == hostevent.MarkedForTerminationEvent
	})).Return(nil)

	return &dynatrace.Client{HostEvent: hostClient}
}

func reconcileAllNodes(t *testing.T, ctrl *Controller, fakeClient client.Client) {
	var nodeList corev1.NodeList
	err := fakeClient.List(t.Context(), &nodeList)

	require.NoError(t, err)

	for _, clusterNode := range nodeList.Items {
		result, err := ctrl.Reconcile(t.Context(), createReconcileRequest(clusterNode.Name))
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
				token.APIKey: []byte(testAPIToken),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oneagent2",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				token.APIKey: []byte(testAPIToken),
			},
		})
}
