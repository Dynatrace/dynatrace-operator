package nodes

import (
	"context"
	"os"
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const testNamespace = "dynatrace"

var testCacheKey = client.ObjectKey{Name: cacheName, Namespace: testNamespace}

func TestNodesReconciler_CreateCache(t *testing.T) {
	fakeClient := createDefaultFakeClient()

	dtClient := &dtclient.MockDynatraceClient{}
	defer mock.AssertExpectationsForObjects(t, dtClient)

	ctrl := createDefaultReconciler(fakeClient, dtClient)

	require.NoError(t, ctrl.reconcileAll())

	var cm corev1.ConfigMap
	require.NoError(t, fakeClient.Get(context.TODO(), testCacheKey, &cm))
	nodesCache := &Cache{Obj: &cm}

	if info, err := nodesCache.Get("node1"); assert.NoError(t, err) {
		assert.Equal(t, "1.2.3.4", info.IPAddress)
		assert.Equal(t, "oneagent1", info.Instance)
	}

	if info, err := nodesCache.Get("node2"); assert.NoError(t, err) {
		assert.Equal(t, "5.6.7.8", info.IPAddress)
		assert.Equal(t, "oneagent2", info.Instance)
	}
}

func TestNodesReconciler_DeleteNode(t *testing.T) {
	fakeClient := createDefaultFakeClient()

	dtClient := createDTMockClient("1.2.3.4", "HOST-42")
	defer mock.AssertExpectationsForObjects(t, dtClient)

	ctrl := createDefaultReconciler(fakeClient, dtClient)

	require.NoError(t, ctrl.reconcileAll())
	require.NoError(t, ctrl.onDeletion("node1"))

	var cm corev1.ConfigMap
	require.NoError(t, fakeClient.Get(context.TODO(), testCacheKey, &cm))
	nodesCache := &Cache{Obj: &cm}

	_, err := nodesCache.Get("node1")
	assert.Equal(t, err, ErrNotFound)

	if info, err := nodesCache.Get("node2"); assert.NoError(t, err) {
		assert.Equal(t, "5.6.7.8", info.IPAddress)
		assert.Equal(t, "oneagent2", info.Instance)
	}
}

func TestNodesReconciler_NodeNotFound(t *testing.T) {
	fakeClient := createDefaultFakeClient()

	dtClient := createDTMockClient("5.6.7.8", "HOST-84")
	defer mock.AssertExpectationsForObjects(t, dtClient)

	ctrl := createDefaultReconciler(fakeClient, dtClient)

	require.NoError(t, ctrl.reconcileAll())
	var node2 corev1.Node
	require.NoError(t, fakeClient.Get(context.TODO(), client.ObjectKey{Name: "node2"}, &node2))
	require.NoError(t, fakeClient.Delete(context.TODO(), &node2))
	require.NoError(t, ctrl.reconcileAll())

	var cm corev1.ConfigMap
	require.NoError(t, fakeClient.Get(context.TODO(), testCacheKey, &cm))
	nodesCache := &Cache{Obj: &cm}

	if info, err := nodesCache.Get("node1"); assert.NoError(t, err) {
		assert.Equal(t, "1.2.3.4", info.IPAddress)
		assert.Equal(t, "oneagent1", info.Instance)
	}

	_, err := nodesCache.Get("node2")
	assert.Equal(t, err, ErrNotFound)
}

func TestNodeReconciler_NodeHasTaint(t *testing.T) {
	fakeClient := createDefaultFakeClient()
	dtClient := createDTMockClient("1.2.3.4", "HOST-42")
	ctrl := createDefaultReconciler(fakeClient, dtClient)

	// Get node 1
	node1 := &corev1.Node{}
	err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: "node1"}, node1)
	assert.NoError(t, err)

	// Add taint that makes it unschedulable
	node1.Spec.Taints = []corev1.Taint{
		{Key: "ToBeDeletedByClusterAutoscaler"},
	}
	err = fakeClient.Update(context.TODO(), node1)
	assert.NoError(t, err)

	// Reconcile all to build cache
	err = ctrl.reconcileAll()
	assert.NoError(t, err)

	// Execute on update which triggers mark for termination
	err = ctrl.onUpdate("node1")
	assert.NoError(t, err)

	// Get node from cache
	c, err := ctrl.getCache()
	assert.NoError(t, err)
	assert.NotNil(t, c)

	node, err := c.Get("node1")
	assert.NoError(t, err)
	assert.NotNil(t, node)

	// Check if LastMarkedForTermination Timestamp is set to current time
	// Added one minute buffer to account for operation times
	now := time.Now().UTC()
	assert.True(t, node.LastMarkedForTermination.Add(time.Minute).After(now))
}

func createDefaultReconciler(fakeClient client.Client, dtClient *dtclient.MockDynatraceClient) *ReconcileNodes {
	return &ReconcileNodes{
		namespace:    testNamespace,
		client:       fakeClient,
		scheme:       scheme.Scheme,
		logger:       zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
		dtClientFunc: dynakube.StaticDynatraceClient(dtClient),
		local:        true,
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

func createDefaultFakeClient() client.Client {
	return fake.NewClient(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}},
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent1", Namespace: testNamespace},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					Instances: map[string]dynatracev1alpha1.OneAgentInstance{"node1": {IPAddress: "1.2.3.4"}},
				},
			},
		},
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent2", Namespace: testNamespace},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					Instances: map[string]dynatracev1alpha1.OneAgentInstance{"node2": {IPAddress: "5.6.7.8"}},
				},
			},
		})
}
