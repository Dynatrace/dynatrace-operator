package v1beta1

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace                 = "test-namespace"
	testName                      = "test-name"
	testUrl                       = "test-url"
	testToken                     = "test-token"
	testCustomPullSecret          = "test-custompullsecret"
	testProxyValue                = "test-proxyvalue"
	testTrustedCAs                = "test-trustedCAs"
	testNetworkZone               = "test-networkzone"
	testPriorityClassName         = "test-priorityclassname"
	testDNSPolicy                 = "test-dnspolicy"
	testActiveGateImage           = "test-activegateimage"
	testStatusOneAgentInstanceKey = "test-instance"
)

func TestConversion_ConvertFrom(t *testing.T) {
	trueVal := true
	time := metav1.Now()
	oldDynakube := &v1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: v1alpha1.DynaKubeSpec{
			APIURL:           testUrl,
			Tokens:           testToken,
			CustomPullSecret: testCustomPullSecret,
			SkipCertCheck:    true,
			Proxy: &v1alpha1.DynaKubeProxy{
				Value: testProxyValue,
			},
			TrustedCAs:  testTrustedCAs,
			NetworkZone: testNetworkZone,
			EnableIstio: true,

			ClassicFullStack: v1alpha1.FullStackSpec{
				Enabled: true,
				NodeSelector: map[string]string{
					"key": "value",
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      "key",
						Operator: "equals",
						Value:    "value",
						Effect:   "effect",
					},
				},
				Resources: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU: *resource.NewScaledQuantity(1, 2),
					},
				},
				Args: []string{
					"arg1",
					"arg2",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "name",
						Value: "value",
					},
				},
				PriorityClassName: testPriorityClassName,
				DNSPolicy:         testDNSPolicy,
				Labels: map[string]string{
					"key": "value",
				},
				UseUnprivilegedMode: &trueVal,
				UseImmutableImage:   true,
			},

			ActiveGate: v1alpha1.ActiveGateSpec{
				Image:      testActiveGateImage,
				AutoUpdate: &trueVal,
			},

			RoutingSpec: v1alpha1.RoutingSpec{
				CapabilityProperties: prepareCapability(),
			},
			KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
				CapabilityProperties: prepareCapability(),
			},
		},
		Status: v1alpha1.DynaKubeStatus{
			Phase:                            "test-phase",
			UpdatedTimestamp:                 time,
			LastAPITokenProbeTimestamp:       &time,
			LastPaaSTokenProbeTimestamp:      &time,
			Tokens:                           "test-tokens",
			LastClusterVersionProbeTimestamp: &time,
			EnvironmentID:                    "test-environment-id",
			Conditions: []metav1.Condition{
				{
					Type:               "type",
					Status:             "status",
					ObservedGeneration: 3,
					LastTransitionTime: time,
					Reason:             "reason",
					Message:            "message",
				},
			},
			ActiveGate: v1alpha1.ActiveGateStatus{
				ImageStatus: v1alpha1.ImageStatus{
					ImageHash:               "test-activegate-imagehash",
					ImageVersion:            "test-activegate-imageversion",
					LastImageProbeTimestamp: &time,
				},
			},
			OneAgent: v1alpha1.OneAgentStatus{
				ImageStatus: v1alpha1.ImageStatus{
					ImageHash:               "test-oneagent-imagehash",
					ImageVersion:            "test-oneagent-imageversion",
					LastImageProbeTimestamp: &time,
				},
				UseImmutableImage: true,
				Version:           "test-oneagent-version",
				Instances: map[string]v1alpha1.OneAgentInstance{
					testStatusOneAgentInstanceKey: {
						PodName:   "test-instance-podname",
						Version:   "test-instance-version",
						IPAddress: "test-instance-ip",
					},
				},
				LastUpdateProbeTimestamp: &time,
			},
		},
	}

	convertedDynakube := &DynaKube{}
	err := convertedDynakube.ConvertFrom(oldDynakube)
	require.NoError(t, err)

	assert.Equal(t, oldDynakube.ObjectMeta.Namespace, convertedDynakube.ObjectMeta.Namespace)
	assert.Equal(t, oldDynakube.ObjectMeta.Name, convertedDynakube.ObjectMeta.Name)

	assert.Equal(t, oldDynakube.Spec.APIURL, convertedDynakube.Spec.APIURL)
	assert.Equal(t, oldDynakube.Spec.Tokens, convertedDynakube.Spec.Tokens)
	assert.Equal(t, oldDynakube.Spec.CustomPullSecret, convertedDynakube.Spec.CustomPullSecret)
	assert.Equal(t, oldDynakube.Spec.SkipCertCheck, convertedDynakube.Spec.SkipCertCheck)
	assert.Equal(t, oldDynakube.Spec.Proxy.ValueFrom, convertedDynakube.Spec.Proxy.ValueFrom)
	assert.Equal(t, oldDynakube.Spec.Proxy.Value, convertedDynakube.Spec.Proxy.Value)
	assert.Equal(t, oldDynakube.Spec.TrustedCAs, convertedDynakube.Spec.TrustedCAs)
	assert.Equal(t, oldDynakube.Spec.NetworkZone, convertedDynakube.Spec.NetworkZone)
	assert.Equal(t, oldDynakube.Spec.EnableIstio, convertedDynakube.Spec.EnableIstio)

	require.NotNil(t, convertedDynakube.Spec.OneAgent.ClassicFullStack)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.NodeSelector, convertedDynakube.Spec.OneAgent.ClassicFullStack.NodeSelector)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.PriorityClassName, convertedDynakube.Spec.OneAgent.ClassicFullStack.PriorityClassName)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Tolerations, convertedDynakube.Spec.OneAgent.ClassicFullStack.Tolerations)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Resources, convertedDynakube.Spec.OneAgent.ClassicFullStack.OneAgentResources)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Args, convertedDynakube.Spec.OneAgent.ClassicFullStack.Args)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.DNSPolicy, convertedDynakube.Spec.OneAgent.ClassicFullStack.DNSPolicy)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Labels, convertedDynakube.Spec.OneAgent.ClassicFullStack.Labels)

	require.NotNil(t, convertedDynakube.Spec.Routing)
	assert.Equal(t, oldDynakube.Spec.ActiveGate.Image, convertedDynakube.Spec.Routing.Image)
	compareCapability(t,
		oldDynakube.Spec.RoutingSpec.CapabilityProperties,
		convertedDynakube.Spec.Routing.CapabilityProperties)

	require.NotNil(t, convertedDynakube.Spec.KubernetesMonitoring)
	compareCapability(t,
		oldDynakube.Spec.KubernetesMonitoringSpec.CapabilityProperties,
		convertedDynakube.Spec.KubernetesMonitoring.CapabilityProperties)

	assert.Len(t, convertedDynakube.Spec.ActiveGate.Capabilities, 0)

	assert.Equal(t, oldDynakube.Status.ActiveGate.ImageHash, convertedDynakube.Status.ActiveGate.ImageHash)
	assert.Equal(t, oldDynakube.Status.ActiveGate.LastImageProbeTimestamp, convertedDynakube.Status.ActiveGate.LastUpdateProbeTimestamp)
	assert.Equal(t, oldDynakube.Status.ActiveGate.ImageVersion, convertedDynakube.Status.ActiveGate.Version)
	assert.Equal(t, oldDynakube.Status.Conditions, convertedDynakube.Status.Conditions)
	assert.Equal(t, oldDynakube.Status.LastAPITokenProbeTimestamp, convertedDynakube.Status.LastAPITokenProbeTimestamp)
	assert.Equal(t, oldDynakube.Status.LastClusterVersionProbeTimestamp, convertedDynakube.Status.LastClusterVersionProbeTimestamp)
	assert.Equal(t, oldDynakube.Status.LastPaaSTokenProbeTimestamp, convertedDynakube.Status.LastPaaSTokenProbeTimestamp)
	assert.Equal(t, oldDynakube.Status.OneAgent.ImageHash, convertedDynakube.Status.OneAgent.ImageHash)

	assert.Len(t, convertedDynakube.Status.OneAgent.Instances, 1)
	oldInstance := oldDynakube.Status.OneAgent.Instances[testStatusOneAgentInstanceKey]
	convertedInstance := convertedDynakube.Status.OneAgent.Instances[testStatusOneAgentInstanceKey]
	assert.Equal(t, oldInstance.IPAddress, convertedInstance.IPAddress)
	assert.Equal(t, oldInstance.PodName, convertedInstance.PodName)

	assert.Equal(t, oldDynakube.Status.OneAgent.LastUpdateProbeTimestamp, convertedDynakube.Status.OneAgent.LastUpdateProbeTimestamp)
	assert.Equal(t, oldDynakube.Status.OneAgent.Version, convertedDynakube.Status.OneAgent.Version)
	assert.Equal(t, string(oldDynakube.Status.Phase), string(convertedDynakube.Status.Phase))
	assert.Equal(t, oldDynakube.Status.Tokens, convertedDynakube.Status.Tokens)
	assert.Equal(t, oldDynakube.Status.UpdatedTimestamp, convertedDynakube.Status.UpdatedTimestamp)
}

func prepareCapability() v1alpha1.CapabilityProperties {
	intVal := int32(3)
	return v1alpha1.CapabilityProperties{
		Enabled:  true,
		Replicas: &intVal,
		Group:    "test-activegate-group",
		CustomProperties: &v1alpha1.DynaKubeValueSource{
			Value: "test-routing-value",
		},
		Resources: corev1.ResourceRequirements{
			Limits: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU: *resource.NewScaledQuantity(1, 1),
			},
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: *resource.NewScaledQuantity(2, 2),
			},
		},
		NodeSelector: map[string]string{
			"key": "value",
		},
		Tolerations: []corev1.Toleration{
			{
				Key:      "key",
				Operator: "operator",
				Value:    "value",
				Effect:   "effect",
			},
		},
		Labels: map[string]string{
			"key": "value",
		},
		Args: []string{
			"arg1",
		},
		Env: []corev1.EnvVar{
			{
				Name:  "name",
				Value: "value",
			},
		},
	}
}

func compareCapability(t *testing.T, expectedCapability v1alpha1.CapabilityProperties, actualCapability CapabilityProperties) {
	assert.Equal(t, expectedCapability.Replicas, actualCapability.Replicas)
	assert.Equal(t, expectedCapability.Group, actualCapability.Group)
	assert.Equal(t, expectedCapability.CustomProperties.ValueFrom, actualCapability.CustomProperties.ValueFrom)
	assert.Equal(t, expectedCapability.CustomProperties.Value, actualCapability.CustomProperties.Value)
	assert.Equal(t, expectedCapability.Resources, actualCapability.Resources)
	assert.Equal(t, expectedCapability.NodeSelector, actualCapability.NodeSelector)
	assert.Equal(t, expectedCapability.Tolerations, actualCapability.Tolerations)
	assert.Equal(t, expectedCapability.Labels, actualCapability.Labels)
	assert.Equal(t, expectedCapability.Args, actualCapability.Args)
	assert.Equal(t, expectedCapability.Env, actualCapability.Env)
}

// todo: convertto
