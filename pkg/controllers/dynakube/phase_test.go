package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
		},
	}

	t.Run("no activegate statefulsets in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("error accessing k8s api -> error", func(t *testing.T) {
		fakeClient := errorClient{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
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
		phase := controller.determineDynaKubePhase(t.Context(), dk)
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
		phase := controller.determineDynaKubePhase(t.Context(), dk)
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
			OneAgent: oneagent.Spec{
				ClassicFullStack: &oneagent.HostInjectSpec{},
			},
		},
	}

	t.Run("no OneAgent pods in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("Error accessing k8s api", func(t *testing.T) {
		fakeClient := errorClient{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Error, phase)
	})
	t.Run("OneAgent daemonsets in cluster not all ready -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient(createDaemonSet(testNamespace, "test-name-oneagent", 3, 2))
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("OneAgent daemonsets in cluster all ready -> running", func(t *testing.T) {
		fakeClient := fake.NewClient(createDaemonSet(testNamespace, "test-name-oneagent", 3, 3))
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Running, phase)
	})
}

func createDaemonSet(namespace, name string, replicas, readyReplicas int32) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: appsv1.DaemonSetStatus{
			CurrentNumberScheduled: replicas,
			NumberReady:            readyReplicas,
		},
	}
}

func TestExtensionsExecutionControllerPhaseChanges(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
		},
	}

	t.Run("no eec statefulsets in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineExtensionsExecutionControllerPhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("error accessing k8s api -> error", func(t *testing.T) {
		fakeClient := errorClient{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineExtensionsExecutionControllerPhase(t.Context(), dk)
		assert.Equal(t, status.Error, phase)
	})
	t.Run("eec pods not ready -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient(createStatefulset(testNamespace, dk.Extensions().GetExecutionControllerStatefulsetName(), 1, 0))

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineExtensionsExecutionControllerPhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("eec deployed -> running", func(t *testing.T) {
		fakeClient := fake.NewClient(createStatefulset(testNamespace, dk.Extensions().GetExecutionControllerStatefulsetName(), 1, 1))

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineExtensionsExecutionControllerPhase(t.Context(), dk)
		assert.Equal(t, status.Running, phase)
	})
}

func TestExtensionsCollectorPhaseChanges(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
		},
	}

	t.Run("no otelc statefulsets in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineExtensionsCollectorPhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("error accessing k8s api -> error", func(t *testing.T) {
		fakeClient := errorClient{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineExtensionsCollectorPhase(t.Context(), dk)
		assert.Equal(t, status.Error, phase)
	})
	t.Run("otelc pods not ready -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient(createStatefulset(testNamespace, dk.OtelCollectorStatefulsetName(), 2, 1))

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineExtensionsCollectorPhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("otelc deployed -> running", func(t *testing.T) {
		fakeClient := fake.NewClient(createStatefulset(testNamespace, dk.OtelCollectorStatefulsetName(), 2, 2))

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineExtensionsCollectorPhase(t.Context(), dk)
		assert.Equal(t, status.Running, phase)
	})
}

func TestExtensionsDatabasesPhaseChanges(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: &extensions.Spec{Databases: []extensions.DatabaseSpec{{}}},
		},
	}

	tests := []struct {
		name        string
		client      client.Client
		expectPhase status.DeploymentPhase
	}{
		{
			"no deployments",
			fake.NewClient(),
			status.Deploying,
		},
		{
			"api error",
			errorClient{},
			status.Error,
		},
		{
			"pods not ready",
			fake.NewClient(createDeployment(dk, 1, 2)),
			status.Deploying,
		},
		{
			"pending deployment reconcile",
			fake.NewClient(func() *appsv1.Deployment {
				deploy := createDeployment(dk, 2, 2)
				deploy.Generation = 2
				deploy.Status.ObservedGeneration = 1

				return deploy
			}()),
			status.Deploying,
		},
		{
			"pods ready",
			fake.NewClient(createDeployment(dk, 2, 2)),
			status.Running,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := &Controller{
				client:    test.client,
				apiReader: test.client,
			}
			phase := controller.determineExtensionsDatabasesPhase(t.Context(), dk)
			assert.Equal(t, test.expectPhase, phase)
		})
	}

	// needs special setup that would complicate the table driven test
	t.Run("ignore deleting deployments", func(t *testing.T) {
		deploy := createDeployment(dk, 1, 2)
		deploy.Finalizers = []string{"keep-me"}
		clt := fake.NewClient(deploy)
		require.NoError(t, clt.Delete(t.Context(), deploy))

		controller := &Controller{
			client:    clt,
			apiReader: clt,
		}
		phase := controller.determineExtensionsDatabasesPhase(t.Context(), dk)
		assert.Equal(t, status.Running, phase)
	})
}

func TestLogAgentPhaseChanges(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			LogMonitoring: &logmonitoring.Spec{},
			Templates: dynakube.TemplatesSpec{
				LogMonitoring: &logmonitoring.TemplateSpec{
					ImageRef: image.Ref{
						Repository: "test",
						Tag:        "test-tag",
					},
				},
			},
		},
	}

	t.Run("no LogAgent pods in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("Error accessing k8s api", func(t *testing.T) {
		fakeClient := errorClient{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Error, phase)
	})
	t.Run("LogAgent daemonsets in cluster not all ready -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient(createDaemonSet(testNamespace, dk.LogMonitoring().GetDaemonSetName(), 3, 2))
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("LogAgent daemonsets in cluster all ready -> running", func(t *testing.T) {
		fakeClient := fake.NewClient(createDaemonSet(testNamespace, dk.LogMonitoring().GetDaemonSetName(), 3, 3))
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Running, phase)
	})
}

func TestKSPMPhaseChanges(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			Kspm: &kspm.Spec{},
			Templates: dynakube.TemplatesSpec{
				KspmNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
					ImageRef: image.Ref{
						Repository: "test",
						Tag:        "test-tag",
					},
				},
			},
		},
	}

	t.Run("no KSPM pods in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("Error accessing k8s api", func(t *testing.T) {
		fakeClient := errorClient{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Error, phase)
	})
	t.Run("KSPM daemonsets in cluster not all ready -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient(createDaemonSet(testNamespace, dk.KSPM().GetDaemonSetName(), 3, 2))
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Deploying, phase)
	})
	t.Run("KSPM daemonsets in cluster all ready -> running", func(t *testing.T) {
		fakeClient := fake.NewClient(createDaemonSet(testNamespace, dk.KSPM().GetDaemonSetName(), 3, 3))
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, status.Running, phase)
	})
}

func TestDynakubePhaseChanges(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				ClassicFullStack: &oneagent.HostInjectSpec{},
			},

			LogMonitoring: &logmonitoring.Spec{},

			Kspm: &kspm.Spec{},

			Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
		},
	}

	agReady := createStatefulset(testNamespace, "test-name-activegate", 1, 1)
	agNotReady := createStatefulset(testNamespace, "test-name-activegate", 1, 0)
	eecReady := createStatefulset(testNamespace, dk.Extensions().GetExecutionControllerStatefulsetName(), 1, 1)
	eecNotReady := createStatefulset(testNamespace, dk.Extensions().GetExecutionControllerStatefulsetName(), 1, 0)
	otelcReady := createStatefulset(testNamespace, dk.OtelCollectorStatefulsetName(), 2, 2)
	otelcNotReady := createStatefulset(testNamespace, dk.OtelCollectorStatefulsetName(), 2, 1)
	oaReady := createDaemonSet(testNamespace, "test-name-oneagent", 3, 3)
	oaNotReady := createDaemonSet(testNamespace, "test-name-oneagent", 3, 2)
	logAgentReady := createDaemonSet(testNamespace, dk.LogMonitoring().GetDaemonSetName(), 3, 3)
	logAgentNotReady := createDaemonSet(testNamespace, dk.LogMonitoring().GetDaemonSetName(), 3, 2)
	kspmReady := createDaemonSet(testNamespace, dk.KSPM().GetDaemonSetName(), 3, 3)
	kspmNotReady := createDaemonSet(testNamespace, dk.KSPM().GetDaemonSetName(), 3, 2)

	tests := []struct {
		clt   client.Client
		phase status.DeploymentPhase
	}{
		{
			clt:   fake.NewClient(agNotReady, oaNotReady, eecNotReady, otelcNotReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agNotReady, oaNotReady, eecNotReady, otelcReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agNotReady, oaNotReady, eecReady, otelcNotReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agNotReady, oaNotReady, eecReady, otelcReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agNotReady, oaReady, eecNotReady, otelcNotReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agNotReady, oaReady, eecNotReady, otelcReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agNotReady, oaReady, eecReady, otelcNotReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agNotReady, oaReady, eecReady, otelcReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agReady, oaNotReady, eecNotReady, otelcNotReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agReady, oaNotReady, eecNotReady, otelcReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agReady, oaNotReady, eecReady, otelcNotReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agReady, oaNotReady, eecReady, otelcReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agReady, oaReady, eecNotReady, otelcNotReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agReady, oaReady, eecNotReady, otelcReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agReady, oaReady, eecReady, otelcNotReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agReady, oaReady, eecReady, otelcReady, logAgentReady, kspmReady),
			phase: status.Running,
		},
		{
			clt:   fake.NewClient(agReady, oaNotReady, eecReady, otelcReady, logAgentNotReady, kspmReady),
			phase: status.Deploying,
		},
		{
			clt:   fake.NewClient(agReady, oaReady, eecReady, otelcReady, logAgentReady, kspmNotReady),
			phase: status.Deploying,
		},
	}

	for i, test := range tests {
		controller := &Controller{
			client:    test.clt,
			apiReader: test.clt,
		}
		phase := controller.determineDynaKubePhase(t.Context(), dk)
		assert.Equal(t, test.phase, phase, "failed", "testcase", i)
	}
}

func createDeployment(dk *dynakube.DynaKube, replicas, readyReplicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: dk.Namespace,
			Labels:    k8slabel.NewAppLabels(k8slabel.DatabaseDatasourceLabel, dk.Name, k8slabel.DatabaseDatasourceLabel, "").BuildLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: readyReplicas,
		},
	}
}
