package builder

import (
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestBuildActiveGateQuery(t *testing.T) {
	t.Run("BuildActiveGateQuery", func(t *testing.T) {
		instance := v1alpha1.ActiveGate{
			Spec: v1alpha1.ActiveGateSpec{
				NetworkZone: "some-network-zone",
			},
		}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Hostname: "some-hostname",
			},
			Status: corev1.PodStatus{
				HostIP: "1.1.1.1",
			}}
		activegateQuery := BuildActiveGateQuery(&instance, &pod)
		assert.NotNil(t, activegateQuery)
		assert.Equal(t, "some-hostname", activegateQuery.Hostname)
		assert.Equal(t, "1.1.1.1", activegateQuery.NetworkAddress)
		assert.Equal(t, "some-network-zone", activegateQuery.NetworkZone)
	})
	t.Run("BuildActiveGateQuery handle nil values", func(t *testing.T) {
		activegateQuery := BuildActiveGateQuery(nil, nil)
		assert.NotNil(t, activegateQuery)
		assert.Equal(t, "", activegateQuery.Hostname)
		assert.Equal(t, "", activegateQuery.NetworkAddress)
		assert.Equal(t, DefaultNetworkZone, activegateQuery.NetworkZone)
	})
	t.Run("BuildActiveGateQuery set network zone", func(t *testing.T) {
		instance := v1alpha1.ActiveGate{
			Spec: v1alpha1.ActiveGateSpec{
				NetworkZone: "some-network-zone",
			},
		}
		activegateQuery := BuildActiveGateQuery(&instance, nil)
		assert.NotNil(t, activegateQuery)
		assert.Equal(t, "", activegateQuery.Hostname)
		assert.Equal(t, "", activegateQuery.NetworkAddress)
		assert.Equal(t, "some-network-zone", activegateQuery.NetworkZone)
	})
}
