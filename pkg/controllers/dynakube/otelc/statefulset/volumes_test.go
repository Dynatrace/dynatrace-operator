package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	otelcconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestVolumes(t *testing.T) {
	t.Run("volume mounts with trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.TrustedCAs = "test-trusted-cas"
		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, trustedCAsVolumeMount())
	})
	t.Run("volume mounts without trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		statefulSet := getStatefulset(t, dk)

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, trustedCAsVolumeMount())
	})
	t.Run("volumes and volume mounts with custom EEC TLS certificate", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = "test-tls-name"
		statefulSet := getStatefulset(t, dk)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      extensionsControllerTLSVolumeName,
			MountPath: customEecTLSCertificatePath,
			ReadOnly:  true,
		}

		expectedVolumes := []corev1.Volume{
			{
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dk.Extensions().GetTokenSecretName(),
						Items: []corev1.KeyToPath{
							{
								Key:  consts.OtelcTokenSecretKey,
								Path: consts.OtelcTokenSecretKey,
							},
						},
						DefaultMode: ptr.To(int32(420)),
					},
				},
			},
			{
				Name: extensionsControllerTLSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "test-tls-name",
						Items: []corev1.KeyToPath{
							{
								Key:  consts.TLSCrtDataName,
								Path: consts.TLSCrtDataName,
							},
						},
					},
				},
			},
		}

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})
	t.Run("volumes with trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.TrustedCAs = "test-trusted-cas"
		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, trustedCAsVolume(dk))
	})
	t.Run("volumes with otelc token", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		statefulSet := getStatefulset(t, dk)

		expectedVolume := corev1.Volume{
			Name: consts.ExtensionsTokensVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dk.Extensions().GetTokenSecretName(),
					Items: []corev1.KeyToPath{
						{
							Key:  consts.OtelcTokenSecretKey,
							Path: consts.OtelcTokenSecretKey,
						},
					},
					DefaultMode: ptr.To(int32(420)),
				},
			},
		}

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})

	t.Run("volumes and volume mounts with telemetry service custom TLS certificate", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.TelemetryIngest = &telemetryingest.Spec{
			TLSRefName: testTelemetryIngestSecret,
		}

		tlsSecret := getTLSSecret(dk.TelemetryIngest().TLSRefName, dk.Namespace, "crt", "key")
		dataIngestToken := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)

		statefulSet := getStatefulset(t, dk, &tlsSecret, &dataIngestToken, &configMap)

		expectedVolume := corev1.Volume{
			Name: customTLSCertVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dk.TelemetryIngest().TLSRefName,
					Items: []corev1.KeyToPath{
						{
							Key:  consts.TLSCrtDataName,
							Path: consts.TLSCrtDataName,
						},
						{
							Key:  consts.TLSKeyDataName,
							Path: consts.TLSKeyDataName,
						},
					},
				},
			},
		}

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      customTLSCertVolumeName,
			MountPath: otelcconsts.CustomTLSCertMountPath,
			ReadOnly:  true,
		}

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
	})
}

func TestVolumesWithTelemetryIngestAndRemoteActiveGate(t *testing.T) {
	t.Run("volumes without trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithTelemetryIngest()
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Volumes, trustedCAsVolume(dk))
	})
	t.Run("volume mounts without trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithTelemetryIngest()
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, trustedCAsVolumeMount())
	})

	t.Run("volumes with trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithTelemetryIngest()
		dk.Spec.TrustedCAs = "test-trusted-cas"
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, trustedCAsVolume(dk))
	})
	t.Run("volume mounts with trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithTelemetryIngest()
		dk.Spec.TrustedCAs = "test-trusted-cas"
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, trustedCAsVolumeMount())
	})
}

func TestVolumesWithTelemetryIngestAndInClusterActiveGate(t *testing.T) {
	t.Run("volumes without trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithTelemetryIngest()
		dk.Spec.ActiveGate = activegate.Spec{
			TLSSecretName: "test-ag-cert",
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.DynatraceAPICapability.DisplayName,
			},
		}
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Volumes, trustedCAsVolume(dk))

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, agCertVolume(dk))
	})
	t.Run("volume mounts without trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithTelemetryIngest()
		dk.Spec.ActiveGate = activegate.Spec{
			TLSSecretName: "test-ag-cert",
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.DynatraceAPICapability.DisplayName,
			},
		}
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, trustedCAsVolumeMount())

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, agCertVolumeMount())
	})

	t.Run("volumes with trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithTelemetryIngest()
		dk.Spec.ActiveGate = activegate.Spec{
			TLSSecretName: "test-ag-cert",
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.DynatraceAPICapability.DisplayName,
			},
		}
		dk.Spec.TrustedCAs = "test-trusted-cas"
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Volumes, trustedCAsVolume(dk))

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, agCertVolume(dk))
	})
	t.Run("volume mounts with trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithTelemetryIngest()
		dk.Spec.ActiveGate = activegate.Spec{
			TLSSecretName: "test-ag-cert",
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.DynatraceAPICapability.DisplayName,
			},
		}
		dk.Spec.TrustedCAs = "test-trusted-cas"
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, trustedCAsVolumeMount())

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, agCertVolumeMount())
	})
}

func TestVolumesWithTelemetryIngestAndExtensionsAndInClusterActiveGate(t *testing.T) {
	t.Run("volumes without trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithExtensionsAndTelemetryIngest()
		dk.Spec.ActiveGate = activegate.Spec{
			TLSSecretName: "test-ag-cert",
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.DynatraceAPICapability.DisplayName,
			},
		}
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Volumes, trustedCAsVolume(dk))

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, agCertVolume(dk))
	})
	t.Run("volume mounts without trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithExtensionsAndTelemetryIngest()
		dk.Spec.ActiveGate = activegate.Spec{
			TLSSecretName: "test-ag-cert",
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.DynatraceAPICapability.DisplayName,
			},
		}
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, trustedCAsVolumeMount())

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, agCertVolumeMount())
	})

	t.Run("volumes with trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithExtensionsAndTelemetryIngest()
		dk.Spec.ActiveGate = activegate.Spec{
			TLSSecretName: "test-ag-cert",
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.DynatraceAPICapability.DisplayName,
			},
		}
		dk.Spec.TrustedCAs = "test-trusted-cas"
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, trustedCAsVolume(dk))

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, agCertVolume(dk))
	})
	t.Run("volume mounts with trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithExtensionsAndTelemetryIngest()
		dk.Spec.ActiveGate = activegate.Spec{
			TLSSecretName: "test-ag-cert",
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.DynatraceAPICapability.DisplayName,
			},
		}
		dk.Spec.TrustedCAs = "test-trusted-cas"
		tokensSecret := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)
		statefulSet := getStatefulset(t, dk, &tokensSecret, &configMap)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, trustedCAsVolumeMount())

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, agCertVolumeMount())
	})
}

func trustedCAsVolume(dk *dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: caCertsVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: dk.Spec.TrustedCAs,
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "certs",
						Path: otelcconsts.TrustedCAsFile,
					},
				},
			},
		},
	}
}

func trustedCAsVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      caCertsVolumeName,
		MountPath: otelcconsts.TrustedCAVolumeMountPath,
		ReadOnly:  true,
	}
}

func agCertVolume(dk *dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: agCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dk.ActiveGate().GetTLSSecretName(),
				Items: []corev1.KeyToPath{
					{
						Key:  dynakube.TLSCertKey,
						Path: otelcconsts.ActiveGateCertFile,
					},
				},
			},
		},
	}
}

func agCertVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      agCertVolumeName,
		MountPath: otelcconsts.ActiveGateTLSCertCAVolumeMountPath,
		ReadOnly:  true,
	}
}

func getTestDynakubeWithTelemetryIngest() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			TelemetryIngest: &telemetryingest.Spec{},
			Templates:       dynakube.TemplatesSpec{OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{}},
		},
	}
}

func getTestDynakubeWithExtensionsAndTelemetryIngest() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions:      &extensions.Spec{&extensions.PrometheusSpec{}},
			TelemetryIngest: &telemetryingest.Spec{},
			Templates:       dynakube.TemplatesSpec{OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{}},
		},
	}
}
