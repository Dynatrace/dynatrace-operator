package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestActiveGatePhaseChanges(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName}},
		},
	}
	t.Run(" no activegate statefulsets in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dynakube)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("activegate pods not ready -> deploying", func(t *testing.T) {
		objects := []client.Object{
			createStatefulset(testNamespace, "test-name-kubemon", 3, 2),
			createStatefulset(testNamespace, "test-name-routing", 3, 2),
			createStatefulset(testNamespace, "test-name-activegate", 3, 2),
			createStatefulset(testNamespace, "test-name-synthetic", 3, 2),
		}

		fakeClient := fake.NewClient(objects...)

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dynakube)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("activegate deployed -> running", func(t *testing.T) {
		objects := []client.Object{
			createStatefulset(testNamespace, "test-name-kubemon", 3, 3),
			createStatefulset(testNamespace, "test-name-routing", 3, 3),
			createStatefulset(testNamespace, "test-name-activegate", 3, 3),
			createStatefulset(testNamespace, "test-name-synthetic", 3, 3),
		}

		fakeClient := fake.NewClient(objects...)

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dynakube)
		assert.Equal(t, status.Running, phase)
	})
}

func createStatefulset(namespace, name string, replicas, readyReplicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:      replicas,
			ReadyReplicas: readyReplicas,
		},
	}
}

func TestOneAgentPhaseChanges(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}
	t.Run("no OneAgent pods in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dynakube)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("OneAgent daemonsets in cluster not all ready -> deploying", func(t *testing.T) {
		objects := []client.Object{
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name-oneagent",
					Namespace: testNamespace,
				},
				Status: appsv1.DaemonSetStatus{
					CurrentNumberScheduled: 3,
					NumberReady:            2,
				},
			},
		}
		fakeClient := fake.NewClient(objects...)
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dynakube)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("OneAgent daemonsets in cluster all ready -> running", func(t *testing.T) {
		objects := []client.Object{
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name-oneagent",
					Namespace: testNamespace,
				},
				Status: appsv1.DaemonSetStatus{
					CurrentNumberScheduled: 3,
					NumberReady:            3,
				},
			},
		}
		fakeClient := fake.NewClient(objects...)
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dynakube)
		assert.Equal(t, status.Running, phase)
	})
}
