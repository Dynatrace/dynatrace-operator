// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package defaulting

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDefault(t *testing.T) {
	const uid = types.UID("abc-123")

	kubeSystem := &corev1.Namespace{Name: system.Namespace, UID: uid}

	t.Run("does nothing if network zone is set", func(t *testing.T) {
		d := &dkDefaulter{apiReader: fake.NewClientBuilder().Build()}
		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{NetworkZone: "zone"}}

		require.NoError(t, d.Default(t.Context(), dk))
		assert.Equal(t, "zone", dk.Spec.NetworkZone)
		assert.Empty(t, dk.Spec.NetworkZoneFallbackMode)
	})

	t.Run("defaults network zone and fallback mode", func(t *testing.T) {
		d := &dkDefaulter{apiReader: fake.NewClientBuilder().WithObjects(kubeSystem).Build()}
		dk := &dynakube.DynaKube{}

		require.NoError(t, d.Default(t.Context(), dk))
		assert.Equal(t, "k8s-abc-123", dk.Spec.NetworkZone)
		assert.Equal(t, dynakube.NZFallbackModeDefault, dk.Spec.NetworkZoneFallbackMode)
	})

	t.Run("keeps existing fallback mode", func(t *testing.T) {
		d := &dkDefaulter{apiReader: fake.NewClientBuilder().WithObjects(kubeSystem).Build()}
		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{NetworkZoneFallbackMode: dynakube.NZFallbackModeAny}}

		require.NoError(t, d.Default(t.Context(), dk))
		assert.Equal(t, "k8s-abc-123", dk.Spec.NetworkZone)
		assert.Equal(t, dynakube.NZFallbackModeAny, dk.Spec.NetworkZoneFallbackMode)
	})

	t.Run("fails if cluster uid can't be determined", func(t *testing.T) {
		d := &dkDefaulter{apiReader: fake.NewClientBuilder().Build()}

		require.Error(t, d.Default(t.Context(), &dynakube.DynaKube{}))
	})
}
