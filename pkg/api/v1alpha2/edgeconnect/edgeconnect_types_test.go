// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImage(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		ec := EdgeConnect{}
		require.Equal(t, defaultEdgeConnectRepository+":"+api.LatestTag, ec.Image())
	})

	t.Run("custom repository and tag", func(t *testing.T) {
		ec := EdgeConnect{Spec: EdgeConnectSpec{ImageRef: image.Ref{Repository: "my.registry/edgeconnect", Tag: "1.2.3"}}}
		require.Equal(t, "my.registry/edgeconnect:1.2.3", ec.Image())
	})

	t.Run("digest takes precedence over tag", func(t *testing.T) {
		ec := EdgeConnect{Spec: EdgeConnectSpec{ImageRef: image.Ref{Repository: "my.registry/edgeconnect", Tag: "1.2.3", Digest: "sha256:abc123"}}}
		require.Equal(t, "my.registry/edgeconnect@sha256:abc123", ec.Image())
	})

	t.Run("digest with default repository", func(t *testing.T) {
		ec := EdgeConnect{Spec: EdgeConnectSpec{ImageRef: image.Ref{Digest: "sha256:abc123"}}}
		require.Equal(t, defaultEdgeConnectRepository+"@sha256:abc123", ec.Image())
	})
}

func TestHostMappings(t *testing.T) {
	t.Run("Get HostMappings", func(t *testing.T) {
		e := EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-edgeconnect",
				Namespace: "test-namespace",
			},
			Status: EdgeConnectStatus{
				KubeSystemUID: "test-kube-system-uid",
			},
		}
		got := e.HostMappings()
		expected := []HostMapping{
			{
				From: "test-edgeconnect.test-namespace.test-kube-system-uid." + kubernetesHostnameSuffix,
				To:   KubernetesDefaultDNS,
			},
		}
		require.Equal(t, expected, got)
	})
}
