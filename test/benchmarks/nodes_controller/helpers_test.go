package nodes_test

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func genNode(index int) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("node-%d", index),
			Labels: map[string]string{
				"kubernetes.io/hostname": fmt.Sprintf("node-%d", index),
			},
		},
		Spec: corev1.NodeSpec{
			Unschedulable: false,
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3.8"),
				corev1.ResourceMemory: resource.MustParse("7Gi"),
			},
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
}

func createNode(tb testing.TB, clt client.Client, i int) *corev1.Node {
	tb.Helper()

	node := genNode(i)

	require.NoError(tb, clt.Create(tb.Context(), node))

	return node
}

func createDynakube(tb testing.TB, clt client.Client, url string, index int, instances map[string]oneagent.Instance) *dynakube.DynaKube {
	tb.Helper()

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("dynakube-%d", index),
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: url,
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
			},
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: oneagent.Status{
				Instances: instances,
			},
		},
	}

	require.NoError(tb, clt.Create(tb.Context(), dk))

	// Update status separately
	dk.Status.OneAgent.Instances = instances
	require.NoError(tb, clt.Status().Update(tb.Context(), dk))

	return dk
}

func createSecret(tb testing.TB, clt client.Client, index int) *corev1.Secret {
	tb.Helper()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("dynakube-%d", index),
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			dtclient.APIToken: []byte(testAPIToken),
		},
	}

	require.NoError(tb, clt.Create(tb.Context(), secret))

	return secret
}

func generateNodeName(i int) string {
	return fmt.Sprintf("node-%d", i)
}

func generateNodeIP(i int) string {
	// Generate IPs in 10.0.0.0/8 range
	octet2 := i / 256
	octet3 := i % 256

	return fmt.Sprintf("10.%d.%d.%d", octet2, octet3, i%10+1)
}

func generateEntityID(i int) string {
	return fmt.Sprintf("HOST-%d", i)
}
