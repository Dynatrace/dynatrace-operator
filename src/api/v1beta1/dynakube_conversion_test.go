package v1beta1

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1"
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
	testOneAgentImage             = "test-oneagent-image"
	testOneAgentVersion           = "test-oneagent-version"
	testPriorityClassName         = "test-priorityclassname"
	testDNSPolicy                 = "test-dnspolicy"
	testActiveGateImage           = "test-activegateimage"
	testStatusOneAgentInstanceKey = "test-instance"
)

func TestConversion_ConvertTFrom_Create(t *testing.T) {
	autoUpdate := true
	oldDynakube := &v1alpha1.DynaKube{
		ObjectMeta: prepareObjectMeta(),
		Spec: v1alpha1.DynaKubeSpec{
			APIURL: testAPIURL,
			Tokens: testToken,

			OneAgent: v1alpha1.OneAgentSpec{
				AutoUpdate: &autoUpdate,
			},

			ClassicFullStack: v1alpha1.FullStackSpec{
				Enabled: true,
			},
			KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
				CapabilityProperties: v1alpha1.CapabilityProperties{
					Enabled: true,
				},
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

	require.NotNil(t, convertedDynakube.Spec.OneAgent.ClassicFullStack)
	assert.Equal(t, oldDynakube.Spec.OneAgent.AutoUpdate, convertedDynakube.Spec.OneAgent.ClassicFullStack.AutoUpdate)
}

func TestConversion_ConvertFrom(t *testing.T) {
	trueVal := true
	time := metav1.Now()
	oldDynakube := &v1alpha1.DynaKube{
		ObjectMeta: prepareObjectMeta(),
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

			OneAgent: v1alpha1.OneAgentSpec{
				Version:    testOneAgentVersion,
				Image:      testOneAgentImage,
				AutoUpdate: &trueVal,
			},

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
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceMemory: *resource.NewScaledQuantity(1, 2),
					},
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
				CapabilityProperties: prepareAlphaCapability(),
			},
			KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
				CapabilityProperties: prepareAlphaCapability(),
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
	assert.Equal(t, oldDynakube.Spec.OneAgent.Image, convertedDynakube.Spec.OneAgent.ClassicFullStack.Image)
	assert.Equal(t, oldDynakube.Spec.OneAgent.Version, convertedDynakube.Spec.OneAgent.ClassicFullStack.Version)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.NodeSelector, convertedDynakube.Spec.OneAgent.ClassicFullStack.NodeSelector)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.PriorityClassName, convertedDynakube.Spec.OneAgent.ClassicFullStack.PriorityClassName)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Tolerations, convertedDynakube.Spec.OneAgent.ClassicFullStack.Tolerations)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Resources, convertedDynakube.Spec.OneAgent.ClassicFullStack.OneAgentResources)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Args, convertedDynakube.Spec.OneAgent.ClassicFullStack.Args)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Env, convertedDynakube.Spec.OneAgent.ClassicFullStack.Env)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.DNSPolicy, convertedDynakube.Spec.OneAgent.ClassicFullStack.DNSPolicy)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Labels, convertedDynakube.Spec.OneAgent.ClassicFullStack.Labels)

	require.NotNil(t, convertedDynakube.Spec.Routing)
	assert.Equal(t, oldDynakube.Spec.ActiveGate.Image, convertedDynakube.Spec.Routing.Image)
	compareAlphaCapability(t,
		oldDynakube.Spec.RoutingSpec.CapabilityProperties,
		convertedDynakube.Spec.Routing.CapabilityProperties)

	require.NotNil(t, convertedDynakube.Spec.KubernetesMonitoring)
	compareAlphaCapability(t,
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

func prepareObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      testName,
	}
}

func prepareAlphaCapability() v1alpha1.CapabilityProperties {
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

func compareAlphaCapability(t *testing.T, expectedCapability v1alpha1.CapabilityProperties, actualCapability CapabilityProperties) {
	assert.Equal(t, expectedCapability.Replicas, actualCapability.Replicas)
	assert.Equal(t, expectedCapability.Group, actualCapability.Group)
	assert.Equal(t, expectedCapability.CustomProperties.ValueFrom, actualCapability.CustomProperties.ValueFrom)
	assert.Equal(t, expectedCapability.CustomProperties.Value, actualCapability.CustomProperties.Value)
	assert.Equal(t, expectedCapability.Resources, actualCapability.Resources)
	assert.Equal(t, expectedCapability.NodeSelector, actualCapability.NodeSelector)
	assert.Equal(t, expectedCapability.Tolerations, actualCapability.Tolerations)
	assert.Equal(t, expectedCapability.Labels, actualCapability.Labels)
	assert.Equal(t, expectedCapability.Env, actualCapability.Env)
}

func TestConversion_ConvertTo(t *testing.T) {
	time := metav1.Now()
	oldDynakube := &DynaKube{
		ObjectMeta: prepareObjectMeta(),
		Spec: DynaKubeSpec{
			APIURL:           testUrl,
			Tokens:           testToken,
			CustomPullSecret: testCustomPullSecret,
			SkipCertCheck:    true,
			Proxy: &DynaKubeProxy{
				Value: testProxyValue,
			},
			TrustedCAs:  testTrustedCAs,
			NetworkZone: testNetworkZone,
			EnableIstio: true,

			OneAgent: OneAgentSpec{
				ClassicFullStack: &HostInjectSpec{
					Image:   testOneAgentImage,
					Version: testOneAgentVersion,
					NodeSelector: map[string]string{
						"key": "value",
					},
					PriorityClassName: "test-priorityclass",
					Tolerations: []corev1.Toleration{
						{
							Key:      "key",
							Operator: "operator",
							Value:    "value",
							Effect:   "effect",
						},
					},
					OneAgentResources: corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceMemory: *resource.NewScaledQuantity(1, 2),
						},
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
					DNSPolicy: testDNSPolicy,
					Labels: map[string]string{
						"key": "value",
					},
				},
			},

			Routing: RoutingSpec{
				Enabled:              true,
				CapabilityProperties: prepareBetaCapability(),
			},

			KubernetesMonitoring: KubernetesMonitoringSpec{
				Enabled:              true,
				CapabilityProperties: prepareBetaCapability(),
			},
		},
		Status: DynaKubeStatus{
			Phase:                            "test-phase",
			UpdatedTimestamp:                 time,
			LastAPITokenProbeTimestamp:       &time,
			LastPaaSTokenProbeTimestamp:      &time,
			Tokens:                           "test-tokens",
			LastClusterVersionProbeTimestamp: &time,
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
			ActiveGate: ActiveGateStatus{},
			OneAgent: OneAgentStatus{
				Instances: map[string]OneAgentInstance{
					testStatusOneAgentInstanceKey: {
						PodName:   "test-instance-podname",
						IPAddress: "test-instance-ip",
					},
				},
			},
		},
	}

	convertedDynakube := &v1alpha1.DynaKube{}
	err := oldDynakube.ConvertTo(convertedDynakube)
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

	require.NotNil(t, convertedDynakube.Spec.ClassicFullStack)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.Image, convertedDynakube.Spec.OneAgent.Image)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.Version, convertedDynakube.Spec.OneAgent.Version)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.NodeSelector, convertedDynakube.Spec.ClassicFullStack.NodeSelector)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.PriorityClassName, convertedDynakube.Spec.ClassicFullStack.PriorityClassName)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.Tolerations, convertedDynakube.Spec.ClassicFullStack.Tolerations)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.OneAgentResources, convertedDynakube.Spec.ClassicFullStack.Resources)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.Args, convertedDynakube.Spec.ClassicFullStack.Args)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.Env, convertedDynakube.Spec.ClassicFullStack.Env)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.DNSPolicy, convertedDynakube.Spec.ClassicFullStack.DNSPolicy)
	assert.Equal(t, oldDynakube.Spec.OneAgent.ClassicFullStack.Labels, convertedDynakube.Spec.ClassicFullStack.Labels)

	require.NotNil(t, convertedDynakube.Spec.ActiveGate)
	assert.Equal(t, oldDynakube.Spec.ActiveGate.Image, convertedDynakube.Spec.ActiveGate.Image)

	require.NotNil(t, convertedDynakube.Spec.RoutingSpec)
	compareBetaCapability(t,
		oldDynakube.Spec.Routing.CapabilityProperties,
		convertedDynakube.Spec.RoutingSpec.CapabilityProperties)

	require.NotNil(t, convertedDynakube.Spec.KubernetesMonitoringSpec)
	compareBetaCapability(t,
		oldDynakube.Spec.KubernetesMonitoring.CapabilityProperties,
		convertedDynakube.Spec.KubernetesMonitoringSpec.CapabilityProperties)

	assert.Equal(t, oldDynakube.Status.ActiveGate.ImageHash, convertedDynakube.Status.ActiveGate.ImageHash)
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

func compareBetaCapability(t *testing.T, expectedCapability CapabilityProperties, actualCapability v1alpha1.CapabilityProperties) {
	assert.Equal(t, expectedCapability.Replicas, actualCapability.Replicas)
	assert.Equal(t, expectedCapability.Group, actualCapability.Group)
	assert.Equal(t, expectedCapability.CustomProperties.ValueFrom, actualCapability.CustomProperties.ValueFrom)
	assert.Equal(t, expectedCapability.CustomProperties.Value, actualCapability.CustomProperties.Value)
	assert.Equal(t, expectedCapability.Resources, actualCapability.Resources)
	assert.Equal(t, expectedCapability.NodeSelector, actualCapability.NodeSelector)
	assert.Equal(t, expectedCapability.Tolerations, actualCapability.Tolerations)
	assert.Equal(t, expectedCapability.Labels, actualCapability.Labels)
	assert.Equal(t, expectedCapability.Env, actualCapability.Env)
}

func prepareBetaCapability() CapabilityProperties {
	intVal := int32(3)
	return CapabilityProperties{
		Replicas: &intVal,
		Group:    "test-activegate-group",
		CustomProperties: &DynaKubeValueSource{
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
		Env: []corev1.EnvVar{
			{
				Name:  "name",
				Value: "value",
			},
		},
	}
}
