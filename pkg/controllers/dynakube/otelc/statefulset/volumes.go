package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	// Volume names and paths
	caCertsVolumeName = "cacerts"

	trustedCAsFile = "rootca.pem"

	customTlsCertVolumeName           = "telemetry-custom-tls"
	customTlsCertMountPath            = "/tls/custom/telemetry"
	extensionsControllerTLSVolumeName = "extensions-controller-tls"
	dataIngestTokenVolumeName         = "api-token"
	dataIngestTokenMountPath          = "/secrets/" + dataIngestTokenVolumeName
)

func setVolumes(dk *dynakube.DynaKube) func(o *appsv1.StatefulSet) {
	var volumes []corev1.Volume

	if dk.IsExtensionsEnabled() {
		volumes = append(
			volumes,
			corev1.Volume{
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
			corev1.Volume{
				Name: extensionsControllerTLSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dk.ExtensionsTLSSecretName(),
						Items: []corev1.KeyToPath{
							{
								Key:  consts.TLSCrtDataName,
								Path: consts.TLSCrtDataName,
							},
						},
					},
				},
			},
		)
	}

	if dk.Spec.TrustedCAs != "" {
		volumes = append(volumes, corev1.Volume{
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
		})
	}

	if dk.TelemetryService().IsEnabled() {
		if dk.TelemetryService().Spec.TlsRefName != "" {
			volumes = append(volumes, corev1.Volume{
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
			})
		}

		volumes = append(volumes, corev1.Volume{
			Name: dataIngestTokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dk.Name,
					Items: []corev1.KeyToPath{
						{
							Key:  dynatrace.DataIngestToken,
							Path: dynatrace.DataIngestToken,
						},
					},
				},
			},
		})
	}

	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.Volumes = volumes
	}
}

func buildContainerVolumeMounts(dk *dynakube.DynaKube) []corev1.VolumeMount {
	vm := []corev1.VolumeMount{
		{Name: consts.ExtensionsTokensVolumeName, ReadOnly: true, MountPath: secretsTokensPath},
	}

	if dk.Spec.TrustedCAs != "" {
		vm = append(vm, corev1.VolumeMount{
			Name:      caCertsVolumeName,
			MountPath: trustedCAVolumeMountPath,
			ReadOnly:  true,
		})
	}

	vm = append(vm, corev1.VolumeMount{
		Name:      extensionsControllerTLSVolumeName,
		MountPath: customEecTLSCertificatePath,
		ReadOnly:  true,
	})

	if dk.TelemetryService().IsEnabled() {
		if dk.TelemetryService().Spec.TlsRefName != "" {
			vm = append(vm, corev1.VolumeMount{
				Name:      customTlsCertVolumeName,
				MountPath: customTlsCertMountPath,
				ReadOnly:  true,
			})
		}

		vm = append(vm, corev1.VolumeMount{
			Name:      dataIngestTokenVolumeName,
			MountPath: dataIngestTokenMountPath,
			ReadOnly:  true,
		})
	}

	return vm
}
