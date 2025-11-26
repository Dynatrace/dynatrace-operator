package dynakube_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testNamespace = "dynatrace"

func TestStatus(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	clt.Create(t.Context(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespace,
			Labels: map[string]string{},
		},
	})

	t.Run("dynakube status conditions are unique", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{},
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{},
		}
		
		conditions.SetDaemonSetCreated(dk.Conditions(), "dummy-type", "dummy-daemonset-name")
		conditions.SetDaemonSetCreated(dk.Conditions(), "dummy-type", "dummy-daemonset-name")

		createDynaKube(t, clt, dk)

		assert.Len(t, *dk.Conditions(), 1)
	})
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
	dk.UpdateStatus(t.Context(), clt)
}