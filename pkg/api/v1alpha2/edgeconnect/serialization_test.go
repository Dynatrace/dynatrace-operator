// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func newEdgeConnect() EdgeConnect {
	return EdgeConnect{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "dynatrace.com/v1alpha2",
			Kind:       "EdgeConnect",
		},
		Spec: EdgeConnectSpec{
			APIServer: "test.dev.dynatracelabs.com",
			OAuth: OAuthSpec{
				ClientSecret: "secret",
				Endpoint:     "https://sso.dynatrace.com/sso/oauth2/token",
				Resource:     "urn:dtaccount:test",
			},
		},
	}
}

func TestSerialization(t *testing.T) {
	t.Run("empty struct fields are dropped", func(t *testing.T) {
		ec := newEdgeConnect()

		data, err := yaml.Marshal(&ec)
		require.NoError(t, err)

		serialized := map[string]any{}
		require.NoError(t, yaml.Unmarshal(data, &serialized))

		spec, _ := serialized["spec"].(map[string]any)
		require.NotNil(t, spec)

		assert.NotContains(t, spec, "imageRef")
		assert.NotContains(t, spec, "resources")
		assert.NotContains(t, serialized, "status")
		assert.NotContains(t, serialized, "metadata")
	})

	t.Run("populated struct fields are still rendered", func(t *testing.T) {
		ec := newEdgeConnect()
		ec.ObjectMeta = metav1.ObjectMeta{Name: "test-edgeconnect", Namespace: "dynatrace"}
		ec.Spec.ImageRef = image.Ref{Repository: "my.registry/edgeconnect", Tag: "1.2.3"}
		ec.Spec.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("100m"),
			},
		}
		ec.Status = EdgeConnectStatus{
			Version:          status.VersionStatus{Version: "1.2.3"},
			UpdatedTimestamp: metav1.Now(),
		}

		data, err := yaml.Marshal(&ec)
		require.NoError(t, err)

		serialized := map[string]any{}
		require.NoError(t, yaml.Unmarshal(data, &serialized))

		spec, _ := serialized["spec"].(map[string]any)
		require.NotNil(t, spec)

		assert.Contains(t, spec, "imageRef")
		assert.Contains(t, spec, "resources")

		statusObj, _ := serialized["status"].(map[string]any)
		require.NotNil(t, statusObj)
		assert.Contains(t, statusObj, "version")
		assert.Contains(t, statusObj, "updatedTimestamp")

		assert.Contains(t, serialized, "metadata")
	})
}
