package edgeconnect_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testEdgeConnectName = "edgeconnect"
	testNamespace       = "dynatrace"

	dummyConditionType    = "dummyType"
	dummyConditionReason  = "dummyReason"
	dummyConditionMessage = "dummyMessage"

	duplicatedConditionErrorMessage = `EdgeConnect.dynatrace.com "edgeconnect" is invalid: status.conditions[1]: Duplicate value: {"type":"dummyType"}`
)

func TestStatus(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	clt.Create(t.Context(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespace,
			Labels: map[string]string{},
		},
	})

	t.Run("can't add duplicated conditions", func(t *testing.T) {
		ec := buildEdgeConnect()
		createEdgeConnect(t, clt, ec)
		dummyCondition := buildCondition()

		// append first condition
		*ec.Conditions() = append(*ec.Conditions(), dummyCondition)
		require.NoError(t, ec.UpdateStatus(t.Context(), clt))

		// check that condition was added
		clt.Get(t.Context(), client.ObjectKeyFromObject(ec), ec)
		require.Len(t, *ec.Conditions(), 1)

		// append duplicated condition
		*ec.Conditions() = append(*ec.Conditions(), dummyCondition)
		require.ErrorContains(t, ec.UpdateStatus(t.Context(), clt), duplicatedConditionErrorMessage)

		// check that condition count is still 1
		clt.Get(t.Context(), client.ObjectKeyFromObject(ec), ec)
		require.Len(t, *ec.Conditions(), 1)
	})
}

func buildEdgeConnect() *edgeconnect.EdgeConnect {
	return &edgeconnect.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testEdgeConnectName,
			Namespace:   testNamespace,
			Annotations: map[string]string{},
		},
		Spec:   edgeconnect.EdgeConnectSpec{},
		Status: edgeconnect.EdgeConnectStatus{},
	}
}

func buildCondition() metav1.Condition {
	return metav1.Condition{
		Type:               dummyConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             dummyConditionReason,
		Message:            dummyConditionMessage,
		LastTransitionTime: metav1.Now(),
	}
}

func createObject(t *testing.T, clt client.Client, obj client.Object) {
	t.Helper()
	require.NoError(t, clt.Create(t.Context(), obj))
	t.Cleanup(func() {
		// t.Context is no longer valid during cleanup
		assert.NoError(t, clt.Delete(context.Background(), obj))
	})
}

func createEdgeConnect(t *testing.T, clt client.Client, ec *edgeconnect.EdgeConnect) {
	status := ec.Status
	createObject(t, clt, ec)
	ec.Status = status
	err := ec.UpdateStatus(t.Context(), clt)
	require.NoError(t, err)
}
