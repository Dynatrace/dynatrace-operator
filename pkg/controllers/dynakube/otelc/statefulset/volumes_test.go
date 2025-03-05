package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/telemetryservice"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	otelcconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestVolumes(t *testing.T) {
	t.Run("volume mounts with trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.TrustedCAs = "test-trusted-cas"
		statefulSet := getStatefulset(t, dk)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      caCertsVolumeName,
			MountPath: otelcconsts.TrustedCAVolumeMountPath,
			ReadOnly:  true,
		}
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
	})
	t.Run("volume mounts without trusted CAs", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		statefulSet := getStatefulset(t, dk)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      caCertsVolumeName,
			MountPath: otelcconsts.TrustedCAVolumeMountPath,
			ReadOnly:  true,
		}

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
	})
	t.Run("volumes and volume mounts with custom EEC TLS certificate", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "test-tls-name"
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
						SecretName: dk.ExtensionsTokenSecretName(),
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

		expectedVolume := corev1.Volume{
			Name: caCertsVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dk.Spec.TrustedCAs,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "certs",
							Path: trustedCAsFile,
						},
					},
				},
			},
		}
		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})
	t.Run("volumes with otelc token", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		statefulSet := getStatefulset(t, dk)

		expectedVolume := corev1.Volume{
			Name: consts.ExtensionsTokensVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dk.ExtensionsTokenSecretName(),
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
		dk.Spec.TelemetryService = &telemetryservice.Spec{
			TlsRefName: testTelemetryServiceSecret,
		}

		tlsSecret := getTLSSecret(dk.TelemetryService().Spec.TlsRefName, dk.Namespace, "crt", "key")
		dataIngestToken := getTokens(dk.Name, dk.Namespace)
		configMap := getConfigConfigMap(dk.Name, dk.Namespace)

		statefulSet := getStatefulset(t, dk, &tlsSecret, &dataIngestToken, &configMap)

		expectedVolume := corev1.Volume{
			Name: customTlsCertVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dk.TelemetryService().Spec.TlsRefName,
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
			Name:      customTlsCertVolumeName,
			MountPath: otelcconsts.CustomTlsCertMountPath,
			ReadOnly:  true,
		}

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
	})
}
