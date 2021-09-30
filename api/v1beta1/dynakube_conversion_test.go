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
	testNamespace         = "test-namespace"
	testName              = "test-name"
	testUrl               = "test-url"
	testToken             = "test-token"
	testCustomPullSecret  = "test-custompullsecret"
	testProxyValueFrom    = "test-proxyvaluefrom"
	testProxyValue        = "test-proxyvalue"
	testTrustedCAs        = "test-trustedCAs"
	testNetworkZone       = "test-networkzone"
	testPriorityClassName = "test-priorityclassname"
	testDNSPolicy         = "test-dnspolicy"
	testActiveGateImage   = "test-activegateimage"
)

func TestConversion_ConvertFrom(t *testing.T) {
	trueVal := true
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
				ValueFrom: testProxyValueFrom,
				Value:     testProxyValue,
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
		},
		Status: v1alpha1.DynaKubeStatus{
			ActiveGate: v1alpha1.ActiveGateStatus{},
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

	assert.NotNil(t, convertedDynakube.Spec.OneAgent.ClassicFullStack)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.NodeSelector, convertedDynakube.Spec.OneAgent.ClassicFullStack.NodeSelector)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.PriorityClassName, convertedDynakube.Spec.OneAgent.ClassicFullStack.PriorityClassName)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Tolerations, convertedDynakube.Spec.OneAgent.ClassicFullStack.Tolerations)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Resources, convertedDynakube.Spec.OneAgent.ClassicFullStack.OneAgentResources)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Args, convertedDynakube.Spec.OneAgent.ClassicFullStack.Args)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.DNSPolicy, convertedDynakube.Spec.OneAgent.ClassicFullStack.DNSPolicy)
	assert.Equal(t, oldDynakube.Spec.ClassicFullStack.Labels, convertedDynakube.Spec.OneAgent.ClassicFullStack.Labels)

	// todo: status, active gate
}

// todo: convertto
