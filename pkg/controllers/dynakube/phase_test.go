package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestActiveGatePhaseChanges(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: dynakube.ActiveGateSpec{Capabilities: []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName}},
		},
	}

	t.Run("no activegate statefulsets in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("error accessing k8s api -> error", func(t *testing.T) {
		fakeClient := errorClient{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dk)
		assert.Equal(t, status.Error, phase)
	})
	t.Run("activegate pods not ready -> deploying", func(t *testing.T) {
		objects := []client.Object{
			createStatefulset(testNamespace, "test-name-kubemon", 3, 2),
			createStatefulset(testNamespace, "test-name-routing", 3, 2),
			createStatefulset(testNamespace, "test-name-activegate", 3, 2),
		}

		fakeClient := fake.NewClient(objects...)

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("activegate deployed -> running", func(t *testing.T) {
		objects := []client.Object{
			createStatefulset(testNamespace, "test-name-kubemon", 3, 3),
			createStatefulset(testNamespace, "test-name-routing", 3, 3),
			createStatefulset(testNamespace, "test-name-activegate", 3, 3),
		}

		fakeClient := fake.NewClient(objects...)

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dk)
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
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				ClassicFullStack: &dynakube.HostInjectSpec{},
			},
		},
	}

	t.Run("no OneAgent pods in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("Error accessing k8s api", func(t *testing.T) {
		fakeClient := errorClient{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(dk)
		assert.Equal(t, status.Error, phase)
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
		phase := controller.determineDynaKubePhase(dk)
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
		phase := controller.determineDynaKubePhase(dk)
		assert.Equal(t, status.Running, phase)
	})
}
