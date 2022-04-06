package validation

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testProxySecret = "proxysecret"

	plainTextProxyUrl = "http://test:test \"#%/<>?[\\]^`{|}pass@proxy-service.dynatrace:3128"
	encodedProxyUrl   = "http://test:test%20%5C%22%23%25%2F%3C%3E%3F%5B%5C%5C%5D%5E%5C%60%7B%7C%7Dpass@proxy-service.dynatrace:3128"
)

func TestConflictingActiveGateConfiguration(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {

		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				Routing: dynatracev1beta1.RoutingSpec{
					Enabled: true,
				},
				KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
					Enabled: true,
				},
			},
		})

		assertAllowedResponseWithWarnings(t, 1, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.RoutingCapability.DisplayName,
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			},
		})

		assertAllowedResponseWithWarnings(t, 3, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.MetricsIngestCapability.DisplayName,
					},
				},
			},
		})
	})
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorConflictingActiveGateSections},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Routing: dynatracev1beta1.RoutingSpec{
						Enabled: true,
					},
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							dynatracev1beta1.RoutingCapability.DisplayName,
						},
					},
				},
			})
	})
}

func TestDuplicateActiveGateCapabilities(t *testing.T) {

	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{fmt.Sprintf(errorDuplicateActiveGateCapability, dynatracev1beta1.RoutingCapability.DisplayName)},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							dynatracev1beta1.RoutingCapability.DisplayName,
							dynatracev1beta1.RoutingCapability.DisplayName,
						},
					},
				},
			})
	})
}

func TestInvalidActiveGateCapabilities(t *testing.T) {

	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{fmt.Sprintf(errorInvalidActiveGateCapability, "invalid-capability")},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							"invalid-capability",
						},
					},
				},
			})
	})
}

func TestMissingActiveGateMemoryLimit(t *testing.T) {
	t.Run(`memory warning in activeGate mode`, func(t *testing.T) {
		assertAllowedResponseWithWarnings(t, 1,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							dynatracev1beta1.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynatracev1beta1.CapabilityProperties{
							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			})
	})
	t.Run(`no memory warning in activeGate mode with memory limit`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							dynatracev1beta1.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynatracev1beta1.CapabilityProperties{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceLimitsMemory: *resource.NewMilliQuantity(1, ""),
								},
							},
						},
					},
				},
			})
	})
}

func TestInvalidActiveGateProxy(t *testing.T) {
	t.Run(`valid proxy url`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     encodedProxyUrl,
						ValueFrom: "",
					},
				},
			})
	})

	t.Run(`invalid proxy url`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorInvalidActiveGateProxyUrl},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     plainTextProxyUrl,
						ValueFrom: "",
					},
				},
			})
	})

	t.Run(`valid proxy secret url`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     "",
						ValueFrom: testProxySecret,
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProxySecret,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"proxy": []byte(encodedProxyUrl),
				},
			})
	})

	t.Run(`missing proxy secret`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorMissingActiveGateProxySecret},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     "",
						ValueFrom: testProxySecret,
					},
				},
			})
	})

	t.Run(`invalid format of proxy secret`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorInvalidProxySecretFormat},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     "",
						ValueFrom: testProxySecret,
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProxySecret,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"invalid-name": []byte(encodedProxyUrl),
				},
			})
	})

	t.Run(`invalid proxy secret url`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorInvalidProxySecretUrl},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Proxy: &dynatracev1beta1.DynaKubeProxy{
						Value:     "",
						ValueFrom: testProxySecret,
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProxySecret,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"proxy": []byte(plainTextProxyUrl),
				},
			})
	})

	t.Run(`invalid proxy secret url - eval`, func(t *testing.T) {
		assert.Equal(t, false, isEvalEscapeNeeded("password"))

		// single quotation mark
		assert.Equal(t, true, isEvalEscapeNeeded("pass\"word"))
		assert.Equal(t, false, isEvalEscapeNeeded("pass\\\"word"))

		// backtick
		assert.Equal(t, true, isEvalEscapeNeeded("pass`word"))
		assert.Equal(t, false, isEvalEscapeNeeded("pass\\`word"))

		// single backslash
		assert.Equal(t, true, isEvalEscapeNeeded("pass\\word"))
		assert.Equal(t, false, isEvalEscapeNeeded("pass\\\\word"))

		// odd number of backslashes
		assert.Equal(t, true, isEvalEscapeNeeded("pass\\\\\\word"))
		assert.Equal(t, false, isEvalEscapeNeeded("pass\\\\\\\\word"))

		// quotation mark, backtick, backslash
		assert.Equal(t, false, isEvalEscapeNeeded("pass\\\"\\`\\\\word"))

		// UTF-8 single character - U+1F600 grinning face
		assert.Equal(t, false, isEvalEscapeNeeded("\xF0\x9F\x98\x80"))
	})
}
