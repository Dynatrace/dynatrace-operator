package k8sentity

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	settingsmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func TestReconcile(t *testing.T) {
	t.Run("no error + no run if no scope in status", func(t *testing.T) {
		clt := settingsmock.NewAPIClient(t)
		dk := createDynaKube()
		dk.Status.Conditions = []metav1.Condition{}

		reconciler := NewReconciler()

		err := reconciler.Reconcile(t.Context(), clt, dk)

		require.NoError(t, err)
		require.Empty(t, dk.Status.KubernetesClusterMEID)

		condition := meta.FindStatusCondition(*dk.Conditions(), meIDConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, k8sconditions.OptionalScopeMissingReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Contains(t, condition.Message, dtclient.TokenScopeSettingsRead)
	})
	t.Run("no error if has valid kube system uuid", func(t *testing.T) {
		clt := settingsmock.NewAPIClient(t)
		clt.EXPECT().
			GetK8sClusterME(anyCtx, "kube-system-uuid").
			Return(settings.K8sClusterME{ID: "KUBERNETES_CLUSTER-0E30FE4BF2007587", Name: "operator test entity 1"}, nil).Once()

		dk := createDynaKube()

		reconciler := NewReconciler()

		err := reconciler.Reconcile(t.Context(), clt, dk)

		require.NoError(t, err)
		require.NotEmpty(t, dk.Status.KubernetesClusterMEID)
	})
	t.Run("no error if no MEs are found", func(t *testing.T) {
		clt := settingsmock.NewAPIClient(t)
		clt.EXPECT().GetK8sClusterME(anyCtx, "kube-system-uuid").Return(settings.K8sClusterME{}, nil)

		dk := createDynaKube()

		reconciler := NewReconciler()

		err := reconciler.Reconcile(t.Context(), clt, dk)

		require.NoError(t, err)
		require.Empty(t, dk.Status.KubernetesClusterMEID)
	})
}

func createDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-dk",
		},
		Spec: dynakube.DynaKubeSpec{
			MetadataEnrichment: metadataenrichment.Spec{
				Enabled: ptr.To(true),
			},
		},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID: "kube-system-uuid",
			Conditions: []metav1.Condition{
				{
					Type:   dtclient.ConditionTypeAPITokenSettingsRead,
					Status: metav1.ConditionTrue,
				},
			},
		},
	}
}
