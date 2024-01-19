package dynakube

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileActiveGate(t *testing.T) {
	ctx := context.Background()
	dynakubeBase := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName}},
		},
	}

	t.Run("no active-gate configured => nothing happens (only call active-gate reconciler)", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		dynakube.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{}
		fakeClient := fake.NewClientWithIndex(dynakube)
		controller := &Controller{
			client:                      fakeClient,
			apiReader:                   fakeClient,
			activegateReconcilerBuilder: activegate.NewReconciler,
		}

		err := controller.reconcileActiveGate(ctx, dynakube, nil, nil, nil, nil)
		require.NoError(t, err)
	})
}
