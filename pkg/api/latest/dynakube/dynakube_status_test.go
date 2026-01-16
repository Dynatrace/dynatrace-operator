package dynakube_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDynakubeName = "dynatrace"
	testNamespace    = "dynatrace"

	dummyConditionType    = "dummyType"
	dummyConditionReason  = "dummyReason"
	dummyConditionMessage = "dummyMessage"

	duplicatedConditionErrorMessage = `DynaKube.dynatrace.com "dynatrace" is invalid: status.conditions[1]: Duplicate value: {"type":"dummyType"}`
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
		dk := buildDynaKube()
		createDynaKube(t, clt, dk)
		dummyCondition := buildCondition()

		// append first condition
		*dk.Conditions() = append(*dk.Conditions(), dummyCondition)
		require.NoError(t, dk.UpdateStatus(t.Context(), clt))

		// check that condition was added
		clt.Get(t.Context(), client.ObjectKeyFromObject(dk), dk)
		require.Len(t, *dk.Conditions(), 1)

		// append duplicated condition
		*dk.Conditions() = append(*dk.Conditions(), dummyCondition)
		require.ErrorContains(t, dk.UpdateStatus(t.Context(), clt), duplicatedConditionErrorMessage)

		// check that condition count is still 1
		clt.Get(t.Context(), client.ObjectKeyFromObject(dk), dk)
		require.Len(t, *dk.Conditions(), 1)
	})
}

func buildDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespace,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
			},
		},
		Status: dynakube.DynaKubeStatus{},
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

func createDynaKube(t *testing.T, clt client.Client, dk *dynakube.DynaKube) {
	status := dk.Status
	createObject(t, clt, dk)
	dk.Status = status
	err := dk.UpdateStatus(t.Context(), clt)
	require.NoError(t, err)
}
