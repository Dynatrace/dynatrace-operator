package kspm

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("no error if not enabled", func(t *testing.T) {
		clt := fake.NewFakeClient()

		dk := createDynaKube(false)

		reconciler := NewReconciler(clt, clt, &dk)
		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
	})

	t.Run("no error if enabled", func(t *testing.T) {
		clt := fake.NewFakeClient()

		dk := createDynaKube(true)

		reconciler := NewReconciler(clt, clt, &dk)
		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
	})
}

func createDynaKube(kspmEnabled bool) dynakube.DynaKube {
	return dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-dk",
		},
		Spec: dynakube.DynaKubeSpec{
			Kspm: kspm.Spec{Enabled: kspmEnabled},
		},
	}
}
