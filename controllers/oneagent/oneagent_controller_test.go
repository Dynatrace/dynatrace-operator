package oneagent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	testClusterID = "test-cluster-id"
	testImage     = "test-image"
	testURL       = "https://test-url"
	testName      = "test-name"
)

var consoleLogger = zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout))

var sampleKubeSystemNS = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "kube-system",
		UID:  "01234-5678-9012-3456",
	},
}

func TestReconcileOneAgent_ReconcileOnEmptyEnvironmentAndDNSPolicy(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	dkSpec := dynatracev1alpha1.DynaKubeSpec{
		APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
		Tokens: dkName,
		ClassicFullStack: dynatracev1alpha1.FullStackSpec{
			Enabled:   true,
			DNSPolicy: corev1.DNSClusterFirstWithHostNet,
			Labels: map[string]string{
				"label_key": "label_value",
			},
		},
	}

	dynakube := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec:       dkSpec,
	}
	fakeClient := fake.NewClient(
		dynakube,
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)

	dtClient := &dtclient.MockDynatraceClient{}

	reconciler := &ReconcileOneAgent{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		instance:  dynakube,
		feature:   ClassicFeature,
		fullStack: &dynakube.Spec.ClassicFullStack,
	}

	rec := utils.Reconciliation{Log: consoleLogger, Instance: dynakube}
	_, err := reconciler.Reconcile(context.TODO(), &rec)
	assert.NoError(t, err)

	dsActual := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: dkName + "-" + reconciler.feature, Namespace: namespace}, dsActual)
	assert.NoError(t, err, "failed to get DaemonSet")
	assert.Equal(t, namespace, dsActual.Namespace, "wrong namespace")
	assert.Equal(t, dkName+"-"+reconciler.feature, dsActual.GetObjectMeta().GetName(), "wrong name")
	assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dsActual.Spec.Template.Spec.DNSPolicy, "wrong policy")
	mock.AssertExpectationsForObjects(t, dtClient)
}

func TestReconcile_PhaseSetCorrectly(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	base := dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				Enabled: true,
			},
		},
	}
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1alpha1.APITokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1alpha1.ReasonTokenReady,
		Message: "Ready",
	})
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1alpha1.PaaSTokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1alpha1.ReasonTokenReady,
		Message: "Ready",
	})

	t.Run("SetPhaseOnError called with different values, object and return value correctly modified", func(t *testing.T) {
		dk := base.DeepCopy()

		res := dk.Status.SetPhaseOnError(nil)
		assert.False(t, res)
		assert.Equal(t, dk.Status.Phase, dynatracev1alpha1.DynaKubePhaseType(""))

		res = dk.Status.SetPhaseOnError(errors.New("dummy error"))
		assert.True(t, res)

		if assert.NotNil(t, dk.Status.Phase) {
			assert.Equal(t, dynatracev1alpha1.Error, dk.Status.Phase)
		}
	})
}

func TestReconcile_TokensSetCorrectly(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"
	base := dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				Enabled: true,
			},
		},
	}
	c := fake.NewClient(
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		fullStack: &base.Spec.ClassicFullStack,
		feature:   ClassicFeature,
		instance:  &base,
	}

	t.Run("reconcileRollout Tokens status set, if empty", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Spec.Tokens = ""
		dk.Status.Tokens = ""
		rec := utils.Reconciliation{Log: consoleLogger, Instance: dk}

		// act
		updateCR, err := reconciler.reconcileRollout(&rec)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, dk.Tokens(), dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})
	t.Run("reconcileRollout Tokens status set, if status has wrong name", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Spec.Tokens = ""
		dk.Status.Tokens = "not the actual name"
		rec := utils.Reconciliation{Log: consoleLogger, Instance: dk}

		// act
		updateCR, err := reconciler.reconcileRollout(&rec)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, dk.Tokens(), dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})

	t.Run("reconcileRollout Tokens status set, not equal to defined name", func(t *testing.T) {
		c = fake.NewClient(
			NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
			sampleKubeSystemNS)

		reconciler := &ReconcileOneAgent{
			client:    c,
			apiReader: c,
			scheme:    scheme.Scheme,
			logger:    consoleLogger,
			instance:  &base,
			feature:   ClassicFeature,
			fullStack: &base.Spec.ClassicFullStack,
		}

		// arrange
		customTokenName := "custom-token-name"
		dk := base.DeepCopy()
		dk.Status.Tokens = dk.Tokens()
		dk.Spec.Tokens = customTokenName
		rec := utils.Reconciliation{Log: consoleLogger, Instance: dk}

		// act
		updateCR, err := reconciler.reconcileRollout(&rec)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, dk.Tokens(), dk.Status.Tokens)
		assert.Equal(t, customTokenName, dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})
}

func TestReconcile_InstancesSet(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"
	base := dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				Enabled: true,
			},
		},
	}

	// arrange
	c := fake.NewClient(
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	oldVersion := "1.186"
	hostIP := "1.2.3.4"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)
	dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{utils.DynatracePaasToken}, nil)
	dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{utils.DynatraceApiToken}, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		instance:  &base,
		fullStack: &base.Spec.ClassicFullStack,
		feature:   ClassicFeature,
	}

	t.Run("reconcileImpl Instances set, if autoUpdate is true", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Status.OneAgent.Version = oldVersion
		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-enabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(dkName, reconciler.feature)
		pod.Spec = newPodSpecForCR(dk, &dynatracev1alpha1.FullStackSpec{}, reconciler.feature, false, consoleLogger, "cluster1")
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = dk.Tokens()

		rec := utils.Reconciliation{Log: consoleLogger, Instance: dk, RequeueAfter: 30 * time.Minute}
		err := reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.instance = dk
		upd, err := reconciler.Reconcile(context.TODO(), &rec)

		assert.NoError(t, err)
		assert.True(t, upd)
		assert.NotNil(t, dk.Status.OneAgent.Instances)
		assert.NotEmpty(t, dk.Status.OneAgent.Instances)
	})

	t.Run("reconcileImpl Instances set, if agentUpdateDisabled is true", func(t *testing.T) {
		dk := base.DeepCopy()
		autoUpdate := false
		dk.Spec.OneAgent.AutoUpdate = &autoUpdate
		dk.Status.OneAgent.Version = oldVersion
		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-disabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(dkName, reconciler.feature)
		pod.Spec = newPodSpecForCR(dk, &dynatracev1alpha1.FullStackSpec{}, reconciler.feature, false, consoleLogger, "cluster1")
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = dk.Tokens()

		rec := utils.Reconciliation{Log: consoleLogger, Instance: dk, RequeueAfter: 30 * time.Minute}
		err := reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.instance = dk
		reconciler.fullStack = &dk.Spec.ClassicFullStack
		upd, err := reconciler.Reconcile(context.TODO(), &rec)

		assert.NoError(t, err)
		assert.True(t, upd)
		assert.NotNil(t, dk.Status.OneAgent.Instances)
		assert.NotEmpty(t, dk.Status.OneAgent.Instances)
	})
}

func NewSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}

func TestUseImmutableImage(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`if image is unset and useImmutableImage is false, default image is used`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				OneAgent:         dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{},
			},
		}
		podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, defaultOneAgentImage, podSpecs.Containers[0].Image)
	})
	t.Run(`if image is set and useImmutableImage is false, set image is used`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				OneAgent: dynatracev1alpha1.OneAgentSpec{
					Image: testImage,
				},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{},
			},
		}
		podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, testImage, podSpecs.Containers[0].Image)
	})
	t.Run(`if image is set and useImmutableImage is true, set image is used`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				OneAgent: dynatracev1alpha1.OneAgentSpec{
					Image: testImage,
				},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
				},
			},
		}
		podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, testImage, podSpecs.Containers[0].Image)
	})
	t.Run(`if image is unset and useImmutableImage is true, image is based on api url`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}
		podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, podSpecs.Containers[0].Image, fmt.Sprintf("%s/linux/oneagent:latest", strings.TrimPrefix(testURL, "https://")))

		instance.Spec.OneAgent.Version = testValue
		podSpecs = newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, podSpecs.Containers[0].Image, fmt.Sprintf("%s/linux/oneagent:%s", strings.TrimPrefix(testURL, "https://"), testValue))
	})
}

func TestCustomPullSecret(t *testing.T) {
	log := logger.NewDTLogger()
	instance := dynatracev1alpha1.DynaKube{
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL:   testURL,
			OneAgent: dynatracev1alpha1.OneAgentSpec{},
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				UseImmutableImage: true,
			},
			CustomPullSecret: testName,
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				UseImmutableImage: true,
			},
		},
	}
	podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.ImagePullSecrets)
	assert.Equal(t, testName, podSpecs.ImagePullSecrets[0].Name)
}

func TestResources(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`minimal cpu request of 100mC is set if no resources specified`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}
		podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)

		hasMinimumCPURequest := resource.NewScaledQuantity(1, -1).Equal(*podSpecs.Containers[0].Resources.Requests.Cpu())
		assert.True(t, hasMinimumCPURequest)
	})
	t.Run(`resource requests and limits set`, func(t *testing.T) {
		cpuRequest := resource.NewScaledQuantity(2, -1)
		cpuLimit := resource.NewScaledQuantity(3, -1)
		memoryRequest := resource.NewScaledQuantity(1, 3)
		memoryLimit := resource.NewScaledQuantity(2, 3)

		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *cpuRequest,
							corev1.ResourceMemory: *memoryRequest,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    *cpuLimit,
							corev1.ResourceMemory: *memoryLimit,
						},
					},
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}

		podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)
		hasCPURequest := cpuRequest.Equal(*podSpecs.Containers[0].Resources.Requests.Cpu())
		hasCPULimit := cpuLimit.Equal(*podSpecs.Containers[0].Resources.Limits.Cpu())
		hasMemoryRequest := memoryRequest.Equal(*podSpecs.Containers[0].Resources.Requests.Memory())
		hasMemoryLimit := memoryLimit.Equal(*podSpecs.Containers[0].Resources.Limits.Memory())

		assert.True(t, hasCPURequest)
		assert.True(t, hasCPULimit)
		assert.True(t, hasMemoryRequest)
		assert.True(t, hasMemoryLimit)
	})
}

func TestArguments(t *testing.T) {
	log := logger.NewDTLogger()
	instance := dynatracev1alpha1.DynaKube{
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL:   testURL,
			OneAgent: dynatracev1alpha1.OneAgentSpec{},
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				UseImmutableImage: true,
				Args:              []string{testValue},
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				UseImmutableImage: true,
			},
		},
	}

	podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.Containers)
	assert.Contains(t, podSpecs.Containers[0].Args, testValue)
}

func TestEnvVars(t *testing.T) {
	log := logger.NewDTLogger()
	reservedVariable := "DT_K8S_NODE_NAME"
	instance := dynatracev1alpha1.DynaKube{
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL:   testURL,
			OneAgent: dynatracev1alpha1.OneAgentSpec{},
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				UseImmutableImage: true,
				Env: []corev1.EnvVar{
					{
						Name:  testName,
						Value: testValue,
					},
					{
						Name:  reservedVariable,
						Value: testValue,
					},
				},
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				UseImmutableImage: true,
			},
		},
	}

	podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.Containers)
	assert.NotEmpty(t, podSpecs.Containers[0].Env)
	assertHasEnvVar(t, testName, testValue, podSpecs.Containers[0].Env)
	assertHasEnvVar(t, reservedVariable, testValue, podSpecs.Containers[0].Env)
}

func assertHasEnvVar(t *testing.T, expectedName string, expectedValue string, envVars []corev1.EnvVar) {
	hasVariable := false
	for _, env := range envVars {
		if env.Name == expectedName {
			hasVariable = true
			assert.Equal(t, expectedValue, env.Value)
		}
	}
	assert.True(t, hasVariable)
}

func TestServiceAccountName(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`has default values`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}

		podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, false, log, testClusterID)
		assert.Equal(t, defaultServiceAccountName, podSpecs.ServiceAccountName)

		instance = dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}
		podSpecs = newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
		assert.Equal(t, defaultUnprivilegedServiceAccountName, podSpecs.ServiceAccountName)
	})
	t.Run(`uses custom value`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage:  true,
					ServiceAccountName: testName,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}
		podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, false, log, testClusterID)
		assert.Equal(t, testName, podSpecs.ServiceAccountName)

		instance = dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage:  true,
					ServiceAccountName: testName,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}

		podSpecs = newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
		assert.Equal(t, testName, podSpecs.ServiceAccountName)
	})
}
