package statefulset

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	otelcConsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestEnvironmentVariables(t *testing.T) {
	t.Run("environment variables with Extensions enabled", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, corev1.EnvVar{Name: envShards, Value: fmt.Sprintf("%d", getReplicas(dk))}, statefulSet.Spec.Template.Spec.Containers[0].Env[0])
		assert.Equal(t, corev1.EnvVar{Name: envPodNamePrefix, Value: dk.Name + "-otel-collector"}, statefulSet.Spec.Template.Spec.Containers[0].Env[1])
		assert.Equal(t, corev1.EnvVar{Name: envPodName, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['statefulset.kubernetes.io/pod-name']",
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[2])
		assert.Equal(t, corev1.EnvVar{Name: envShardId, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['apps.kubernetes.io/pod-index']",
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[3])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPgrpcPort, Value: defaultOLTPgrpcPort}, statefulSet.Spec.Template.Spec.Containers[0].Env[4])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPhttpPort, Value: defaultOLTPhttpPort}, statefulSet.Spec.Template.Spec.Containers[0].Env[5])
		assert.Equal(t, corev1.EnvVar{Name: envK8sClusterName, Value: dk.Name}, statefulSet.Spec.Template.Spec.Containers[0].Env[6])
		assert.Equal(t, corev1.EnvVar{Name: envK8sClusterUid, Value: dk.Status.KubeSystemUUID}, statefulSet.Spec.Template.Spec.Containers[0].Env[7])
		assert.Equal(t, corev1.EnvVar{Name: envDTentityK8sCluster, Value: dk.Status.KubernetesClusterMEID}, statefulSet.Spec.Template.Spec.Containers[0].Env[8])
		assert.Equal(t, corev1.EnvVar{Name: envEECDStoken, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dk.ExtensionsTokenSecretName()},
				Key:                  consts.OtelcTokenSecretKey,
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[9])
		assert.Equal(t, corev1.EnvVar{Name: envCertDir, Value: customEecTLSCertificatePath}, statefulSet.Spec.Template.Spec.Containers[0].Env[10])
	})
	t.Run("environment variables with trustedCA", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.TrustedCAs = "test-trusted-ca"

		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envTrustedCAs, Value: otelcConsts.TrustedCAVolumePath})
	})
	t.Run("environment variables with custom EEC TLS certificate", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "test-tls-ca"

		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envEECcontrollerTLS, Value: customEecTLSCertificateFullPath})
	})
	t.Run("environment variables for open signal configuration", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.TelemetryIngest = &telemetryingest.Spec{}

		dataIngestToken := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &dataIngestToken, &configMap)
		assert.Len(t, statefulSet.Spec.Template.Spec.Containers[0].Env, 12)

		assert.Equal(t, corev1.EnvVar{Name: envShards, Value: fmt.Sprintf("%d", getReplicas(dk))}, statefulSet.Spec.Template.Spec.Containers[0].Env[0])
		assert.Equal(t, corev1.EnvVar{Name: envPodNamePrefix, Value: dk.Name + "-otel-collector"}, statefulSet.Spec.Template.Spec.Containers[0].Env[1])
		assert.Equal(t, corev1.EnvVar{Name: envPodName, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['statefulset.kubernetes.io/pod-name']",
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[2])
		assert.Equal(t, corev1.EnvVar{Name: envShardId, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['apps.kubernetes.io/pod-index']",
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[3])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPgrpcPort, Value: defaultOLTPgrpcPort}, statefulSet.Spec.Template.Spec.Containers[0].Env[4])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPhttpPort, Value: defaultOLTPhttpPort}, statefulSet.Spec.Template.Spec.Containers[0].Env[5])
		assert.Equal(t, corev1.EnvVar{Name: envK8sClusterName, Value: dk.Name}, statefulSet.Spec.Template.Spec.Containers[0].Env[6])
		assert.Equal(t, corev1.EnvVar{Name: envK8sClusterUid, Value: dk.Status.KubeSystemUUID}, statefulSet.Spec.Template.Spec.Containers[0].Env[7])
		assert.Equal(t, corev1.EnvVar{Name: envDTentityK8sCluster, Value: dk.Status.KubernetesClusterMEID}, statefulSet.Spec.Template.Spec.Containers[0].Env[8])

		assert.Equal(t, corev1.EnvVar{Name: envDTendpoint, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: otelcConsts.TelemetryApiCredentialsSecretName},
				Key:                  envDTendpoint,
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[9])
		assert.Equal(t, corev1.EnvVar{Name: envMyPodIP, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[10])

		assert.Equal(t, corev1.EnvVar{Name: otelcConsts.EnvDataIngestToken, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dk.Tokens()},
				Key:                  dynatrace.DataIngestToken,
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[11])
	})
}

func TestProxyEnvs(t *testing.T) {
	const testProxySecretName = "test-proxy-secret"

	const testProxyValue = "http://test.proxy.com:8888"

	testActiveGate := activegate.Spec{
		CapabilityProperties: activegate.CapabilityProperties{},
		Capabilities:         []activegate.CapabilityDisplayName{"otlp-ingest"},
		UseEphemeralVolume:   false,
	}

	tests := []struct {
		name            string
		extensions      *dynakube.ExtensionsSpec
		telemetryIngest *telemetryingest.Spec
		activeGate      *activegate.Spec
		proxy           *value.Source

		expectedNoProxy string
	}{
		{
			name:            "extensions without proxy",
			extensions:      &dynakube.ExtensionsSpec{},
			telemetryIngest: nil,
			proxy:           nil,
			expectedNoProxy: "",
		},
		{
			name:            "extensions with proxy secret",
			extensions:      &dynakube.ExtensionsSpec{},
			telemetryIngest: nil,
			proxy: &value.Source{
				ValueFrom: testProxySecretName,
			},
			expectedNoProxy: "dynakube-extensions-controller.dynatrace,dynakube-activegate.dynatrace",
		},
		{
			name:            "extensions with proxy value",
			extensions:      &dynakube.ExtensionsSpec{},
			telemetryIngest: nil,
			proxy: &value.Source{
				Value: testProxyValue,
			},
			expectedNoProxy: "dynakube-extensions-controller.dynatrace,dynakube-activegate.dynatrace",
		},
		{
			name:            "telemetryIngest, public AG, without proxy",
			extensions:      nil,
			telemetryIngest: &telemetryingest.Spec{},
			activeGate:      nil,
			proxy:           nil,
			expectedNoProxy: "",
		},
		{
			name:            "telemetryIngest, public AG, with proxy secret",
			extensions:      nil,
			telemetryIngest: &telemetryingest.Spec{},
			activeGate:      nil,
			proxy: &value.Source{
				ValueFrom: testProxySecretName,
			},
			expectedNoProxy: "",
		},
		{
			name:            "telemetryIngest, public AG, with proxy value",
			extensions:      nil,
			telemetryIngest: &telemetryingest.Spec{},
			activeGate:      nil,
			proxy: &value.Source{
				Value: testProxyValue,
			},
			expectedNoProxy: "",
		},
		{
			name:            "telemetryIngest, local AG, without proxy",
			extensions:      nil,
			telemetryIngest: &telemetryingest.Spec{},
			activeGate:      &testActiveGate,
			proxy:           nil,
			expectedNoProxy: "",
		},
		{
			name:            "telemetryIngest, local AG, with proxy secret",
			extensions:      nil,
			telemetryIngest: &telemetryingest.Spec{},
			activeGate:      &testActiveGate,
			proxy: &value.Source{
				ValueFrom: testProxySecretName,
			},
			expectedNoProxy: "dynakube-activegate.dynatrace",
		},
		{
			name:            "telemetryIngest, local AG, with proxy value",
			extensions:      nil,
			telemetryIngest: &telemetryingest.Spec{},
			activeGate:      &testActiveGate,
			proxy: &value.Source{
				Value: testProxyValue,
			},
			expectedNoProxy: "dynakube-activegate.dynatrace",
		},
		{
			name:            "telemetryIngest, extensions, private AG, without proxy",
			extensions:      &dynakube.ExtensionsSpec{},
			telemetryIngest: &telemetryingest.Spec{},
			activeGate:      nil,
			proxy:           nil,
			expectedNoProxy: "",
		},
		{
			name:            "telemetryIngest, extensions, private AG, with proxy secret",
			extensions:      &dynakube.ExtensionsSpec{},
			telemetryIngest: &telemetryingest.Spec{},
			activeGate:      nil,
			proxy: &value.Source{
				ValueFrom: testProxySecretName,
			},
			expectedNoProxy: "dynakube-extensions-controller.dynatrace,dynakube-activegate.dynatrace",
		},
		{
			name:            "telemetryIngest, extensions, private AG, with proxy value",
			extensions:      &dynakube.ExtensionsSpec{},
			telemetryIngest: &telemetryingest.Spec{},
			activeGate:      nil,
			proxy: &value.Source{
				Value: testProxyValue,
			},
			expectedNoProxy: "dynakube-extensions-controller.dynatrace,dynakube-activegate.dynatrace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dk := getTestDynakube()
			dk.Spec.Extensions = tt.extensions
			dk.Spec.TelemetryIngest = tt.telemetryIngest
			dk.Spec.Proxy = tt.proxy

			if tt.activeGate != nil {
				dk.Spec.ActiveGate = *tt.activeGate
			}

			dataIngestToken := getTokens(dk.Name, dk.Namespace)
			configMap := getConfigConfigMap(dk.Name, dk.Namespace)
			statefulSet := getStatefulset(t, dk, &dataIngestToken, &configMap)

			switch {
			case tt.proxy == nil:
				{
					assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envHttpsProxy})
				}
			case tt.proxy.ValueFrom != "":
				{
					assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
						Name: envHttpsProxy,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: tt.proxy.ValueFrom},
								Key:                  dynakube.ProxyKey,
							},
						},
					})
					assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
						Name: envHttpProxy,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: tt.proxy.ValueFrom},
								Key:                  dynakube.ProxyKey,
							},
						},
					})
				}
			default:
				{
					assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
						Name:  envHttpsProxy,
						Value: tt.proxy.Value,
					})
					assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
						Name:  envHttpProxy,
						Value: tt.proxy.Value,
					})
				}
			}

			if tt.proxy != nil {
				assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envNoProxy, Value: tt.expectedNoProxy})
			} else {
				assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envNoProxy})
			}
		})
	}
}
