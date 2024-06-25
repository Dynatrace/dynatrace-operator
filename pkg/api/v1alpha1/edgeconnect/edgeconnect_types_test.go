package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/edgeconnect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
		expected := []edgeconnect.HostMapping{
			{
				From: "test-edgeconnect.test-namespace.test-kube-system-uid." + kubernetesHostnameSuffix,
				To:   kubernetesDefaultDNS,
			},
		}
		require.EqualValues(t, expected, got)
	})
}
