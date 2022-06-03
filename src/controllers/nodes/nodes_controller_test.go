package nodes

import (
	"context"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const testNamespace = "dynatrace"

var testCacheKey = client.ObjectKey{Name: cacheName, Namespace: testNamespace}

func TestReconcile(t *testing.T) {
	ctx := context.TODO()
	t.Run("Create cache", func(t *testing.T) {
		fakeClient := createDefaultFakeClient()

		dtClient := &dtclient.MockDynatraceClient{}
		defer mock.AssertExpectationsForObjects(t, dtClient)

		ctrl := createDefaultReconciler(fakeClient, dtClient)
		result, err := ctrl.Reconcile(ctx, createReconcileRequest("node1"))
		assert.Nil(t, err)
		assert.NotNil(t, result)

		var cm corev1.ConfigMap
		require.NoError(t, fakeClient.Get(ctx, testCacheKey, &cm))
		nodesCache := &Cache{Obj: &cm}

		if info, err := nodesCache.Get("node1"); assert.NoError(t, err) {
			assert.Equal(t, "1.2.3.4", info.IPAddress)
			assert.Equal(t, "oneagent1", info.Instance)
		}
	})

	t.Run("Delete node", func(t *testing.T) {
		fakeClient := createDefaultFakeClient()

		dtClient := createDTMockClient("1.2.3.4", "HOST-42")
		defer mock.AssertExpectationsForObjects(t, dtClient)

		ctrl := createDefaultReconciler(fakeClient, dtClient)
		reconcileAllNodes(t, ctrl, fakeClient)
		assert.NoError(t, ctrl.reconcileNodeDeletion(ctx, "node1"))

		var cm corev1.ConfigMap
		require.NoError(t, fakeClient.Get(ctx, testCacheKey, &cm))
		nodesCache := &Cache{Obj: &cm}

		_, err := nodesCache.Get("node1")
		assert.Equal(t, err, ErrNotFound)

		if info, err := nodesCache.Get("node2"); assert.NoError(t, err) {
			assert.Equal(t, "5.6.7.8", info.IPAddress)
			assert.Equal(t, "oneagent2", info.Instance)
		}
	})
	t.Run("Node not found", func(t *testing.T) {
		fakeClient := createDefaultFakeClient()

		dtClient := createDTMockClient("5.6.7.8", "HOST-84")
		defer mock.AssertExpectationsForObjects(t, dtClient)

		ctrl := createDefaultReconciler(fakeClient, dtClient)

		reconcileAllNodes(t, ctrl, fakeClient)

		var node2 corev1.Node
		require.NoError(t, fakeClient.Get(context.TODO(), client.ObjectKey{Name: "node2"}, &node2))
		require.NoError(t, fakeClient.Delete(context.TODO(), &node2))

		assert.NoError(t, ctrl.reconcileNodeDeletion(ctx, "node2"))

		var cm corev1.ConfigMap
		require.NoError(t, fakeClient.Get(context.TODO(), testCacheKey, &cm))
		nodesCache := &Cache{Obj: &cm}

		if info, err := nodesCache.Get("node1"); assert.NoError(t, err) {
			assert.Equal(t, "1.2.3.4", info.IPAddress)
			assert.Equal(t, "oneagent1", info.Instance)
		}

		_, err := nodesCache.Get("node2")
		assert.Equal(t, err, ErrNotFound)
	})
	t.Run("Node has taint", func(t *testing.T) {
		fakeClient := createDefaultFakeClient()
		dtClient := createDTMockClient("1.2.3.4", "HOST-42")
		ctrl := createDefaultReconciler(fakeClient, dtClient)

		// Get node 1
		node1 := &corev1.Node{}
		err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: "node1"}, node1)
		assert.NoError(t, err)

		reconcileAllNodes(t, ctrl, fakeClient)
		// Add taint that makes it unschedulable
		node1.Spec.Taints = []corev1.Taint{
			{Key: "ToBeDeletedByClusterAutoscaler"},
		}
		err = fakeClient.Update(context.TODO(), node1)
		assert.NoError(t, err)

		result, err := ctrl.Reconcile(context.TODO(), createReconcileRequest("node1"))
		assert.NotNil(t, result)
		assert.NoError(t, err)

		// Get node from cache
		c, err := ctrl.getCache(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, c)

		node, err := c.Get("node1")
		assert.NoError(t, err)
		assert.NotNil(t, node)

		// Check if LastMarkedForTermination Timestamp is set to current time
		// Added one minute buffer to account for operation times
		now := time.Now().UTC()
		assert.True(t, node.LastMarkedForTermination.Add(time.Minute).After(now))
	})

	t.Run("Server error when removing node", func(t *testing.T) {
		fakeClient := createDefaultFakeClient()

		dtClient := &dtclient.MockDynatraceClient{}
		dtClient.On("GetEntityIDForIP", mock.Anything).Return("", ErrNotFound)

		ctrl := createDefaultReconciler(fakeClient, dtClient)

		reconcileAllNodes(t, ctrl, fakeClient)

		assert.Error(t, ctrl.reconcileNodeDeletion(ctx, "node1"))
	})
}

func createReconcileRequest(nodeName string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: nodeName},
	}
}

func createDefaultReconciler(fakeClient client.Client, dtClient *dtclient.MockDynatraceClient) *NodesController {
	return &NodesController{
		client:       fakeClient,
		apiReader:    fakeClient,
		scheme:       scheme.Scheme,
		dtClientFunc: dynakube.StaticDynatraceClient(dtClient),
		podNamespace: testNamespace,
		runLocal:     true,
	}
}

func createDTMockClient(ip, host string) *dtclient.MockDynatraceClient {
	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetEntityIDForIP", ip).Return(host, nil)
	dtClient.On("SendEvent", mock.MatchedBy(func(e *dtclient.EventData) bool {
		return e.EventType == "MARKED_FOR_TERMINATION"
	})).Return(nil)
	return dtClient
}

func reconcileAllNodes(t *testing.T, ctrl *NodesController, fakeClient client.Client) {
	var nodeList corev1.NodeList
	fakeClient.List(context.TODO(), &nodeList)

	for _, clusterNode := range nodeList.Items {
		result, err := ctrl.Reconcile(context.TODO(), createReconcileRequest(clusterNode.Name))
		assert.Nil(t, err)
		assert.NotNil(t, result)
	}
}

func createDefaultFakeClient() client.Client {
	return fake.NewClient(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}},
		&dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent1", Namespace: testNamespace},
			Status: dynatracev1beta1.DynaKubeStatus{
				OneAgent: dynatracev1beta1.OneAgentStatus{
					Instances: map[string]dynatracev1beta1.OneAgentInstance{"node1": {IPAddress: "1.2.3.4"}},
				},
			},
		},
		&dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent2", Namespace: testNamespace},
			Status: dynatracev1beta1.DynaKubeStatus{
				OneAgent: dynatracev1beta1.OneAgentStatus{
					Instances: map[string]dynatracev1beta1.OneAgentInstance{"node2": {IPAddress: "5.6.7.8"}},
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oneagent1",
				Namespace: testNamespace,
			},
		})
}
