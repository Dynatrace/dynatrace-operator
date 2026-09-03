// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/stretchr/testify/require"
)

func Test_publicRegistryOverrideWithCustomImage(t *testing.T) {
	t.Run("accept when neither field is set", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{},
		}
		require.Empty(t, publicRegistryOverrideWithCustomImage(t.Context(), nil, ec))
	})
	t.Run("accept when only publicRegistryOverride is set", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				PublicRegistryOverride: "my.registry.example.com",
			},
		}
		require.Empty(t, publicRegistryOverrideWithCustomImage(t.Context(), nil, ec))
	})
	t.Run("accept when only custom imageRef is set", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ImageRef: image.Ref{
					Repository: "my.registry.example.com/edgeconnect",
				},
			},
		}
		require.Empty(t, publicRegistryOverrideWithCustomImage(t.Context(), nil, ec))
	})
	t.Run("reject when both publicRegistryOverride and custom imageRef are set", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				PublicRegistryOverride: "my.registry.example.com",
				ImageRef: image.Ref{
					Repository: "my.registry.example.com/edgeconnect",
				},
			},
		}
		require.Equal(t, errorPublicRegistryOverrideWithCustomImage, publicRegistryOverrideWithCustomImage(t.Context(), nil, ec))
	})
}
