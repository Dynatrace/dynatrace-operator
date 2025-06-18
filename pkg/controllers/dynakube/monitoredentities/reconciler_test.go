package monitoredentities

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("no error if not enabled", func(t *testing.T) {
		clt := dtclientmock.NewClient(t)
		clt.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), "kube-system-uuid").Return([]dtclient.MonitoredEntity{{EntityID: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity 1", LastSeenTms: 1639483869085}}, nil)

		dk := createDynaKube()
		dk.Spec.MetadataEnrichment.Enabled = ptr.To(false)

		reconciler := NewReconciler(clt, &dk)

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
	})
	t.Run("no error if enabled and has valid kube system uuid", func(t *testing.T) {
		clt := dtclientmock.NewClient(t)
		clt.On("GetMonitoredEntitiesForKubeSystemUUID",
			mock.AnythingOfType("context.backgroundCtx"), "kube-system-uuid").Return([]dtclient.MonitoredEntity{{EntityID: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity 1", LastSeenTms: 1639483869085}}, nil)

		dk := createDynaKube()

		reconciler := NewReconciler(clt, &dk)

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		require.NotEmpty(t, dk.Status.KubernetesClusterMEID)
	})
	t.Run("no error if no MEs are found", func(t *testing.T) {
		clt := dtclientmock.NewClient(t)
		clt.On("GetMonitoredEntitiesForKubeSystemUUID",
			mock.AnythingOfType("context.backgroundCtx"), "kube-system-uuid").Return([]dtclient.MonitoredEntity{}, nil)

		dk := createDynaKube()

		reconciler := NewReconciler(clt, &dk)

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		require.Empty(t, dk.Status.KubernetesClusterMEID)
	})
}

func createDynaKube() dynakube.DynaKube {
	return dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-dk",
		},
		Spec: dynakube.DynaKubeSpec{
			MetadataEnrichment: dynakube.MetadataEnrichment{
				Enabled: ptr.To(true),
			},
		},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID: "kube-system-uuid",
		},
	}
}
