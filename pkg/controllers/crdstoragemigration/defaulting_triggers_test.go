// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package crdstoragemigration

import (
	"context"
	"slices"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestTriggerDefaulting(t *testing.T) {
	newDK := func(name, networkZone string) *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
			Spec:       dynakube.DynaKubeSpec{NetworkZone: networkZone},
		}
	}

	run := func(t *testing.T) []string {
		t.Helper()

		var updated []string

		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				updated = append(updated, obj.GetName())

				return c.Update(ctx, obj, opts...)
			},
		}, newDK("empty", ""), newDK("set", "zone"))

		require.NoError(t, TriggerDefaulting(t.Context(), clt, testNamespace))

		return updated
	}

	t.Run("updates only DynaKubes with an empty network zone", func(t *testing.T) {
		assert.Equal(t, []string{"empty"}, run(t))
	})

	t.Run("updates a DynaKube only once if multiple triggers apply", func(t *testing.T) {
		original := defaultingTriggers
		t.Cleanup(func() { defaultingTriggers = original })

		defaultingTriggers = append(slices.Clone(original), defaultingTrigger{
			name:            "second",
			needsDefaulting: func(*dynakube.DynaKube) bool { return true },
		})

		assert.Equal(t, []string{"empty", "set"}, run(t))
	})
}
