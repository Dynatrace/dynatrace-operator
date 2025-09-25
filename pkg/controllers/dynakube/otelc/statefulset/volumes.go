package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/configuration"
	otelcconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	// Volume names and paths
	caCertsVolumeName = "cacerts"
	agCertVolumeName  = "agcert"

	customTLSCertVolumeName            = "telemetry-ingest-custom-tls"
	extensionsControllerTLSVolumeName  = "extensions-controller-tls"
	telemetryCollectorConfigVolumeName = "telemetry-collector-config"
	telemetryCollectorConfigPath       = "/config"
)

func setVolumes(dk *dynakube.DynaKube) func(o *appsv1.StatefulSet) {
	var volumes []corev1.Volume

	if ext := dk.Extensions(); ext.IsEnabled() {
		volumes = append(
			volumes,
			corev1.Volume{
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: ext.GetTokenSecretName(),
						Items: []corev1.KeyToPath{
							{
								Key:  consts.DatasourceTokenSecretKey,
								Path: consts.DatasourceTokenSecretKey,
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
						SecretName: dk.Extensions().GetTLSSecretName(),
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

	if isTrustedCAsVolumeNeeded(dk) {
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
							Path: otelcconsts.TrustedCAsFile,
						},
					},
				},
			},
		})
	}

	if dk.TelemetryIngest().IsEnabled() {
		if dk.IsAGCertificateNeeded() {
			volumes = append(volumes, corev1.Volume{
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
			})
		}

		if dk.TelemetryIngest().TLSRefName != "" {
			volumes = append(volumes, corev1.Volume{
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
			})
		}

		volumes = append(volumes, corev1.Volume{
			Name: telemetryCollectorConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configuration.GetConfigMapName(dk.Name),
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
	var vm []corev1.VolumeMount

	if dk.Extensions().IsEnabled() {
		vm = append(
			vm,
			corev1.VolumeMount{
				Name: consts.ExtensionsTokensVolumeName, ReadOnly: true, MountPath: secretsTokensPath,
			},
			corev1.VolumeMount{
				Name:      extensionsControllerTLSVolumeName,
				MountPath: customEecTLSCertificatePath,
				ReadOnly:  true,
			},
		)
	}

	if isTrustedCAsVolumeNeeded(dk) {
		vm = append(vm, corev1.VolumeMount{
			Name:      caCertsVolumeName,
			MountPath: otelcconsts.TrustedCAVolumeMountPath,
			ReadOnly:  true,
		})
	}

	if dk.TelemetryIngest().IsEnabled() {
		if dk.IsAGCertificateNeeded() {
			vm = append(vm, corev1.VolumeMount{
				Name:      agCertVolumeName,
				MountPath: otelcconsts.ActiveGateTLSCertCAVolumeMountPath,
				ReadOnly:  true,
			})
		}

		if dk.TelemetryIngest().TLSRefName != "" {
			vm = append(vm, corev1.VolumeMount{
				Name:      customTLSCertVolumeName,
				MountPath: otelcconsts.CustomTLSCertMountPath,
				ReadOnly:  true,
			})
		}

		vm = append(vm, corev1.VolumeMount{
			Name:      telemetryCollectorConfigVolumeName,
			MountPath: telemetryCollectorConfigPath,
			ReadOnly:  true,
		})
	}

	return vm
}

func isTrustedCAsVolumeNeeded(dk *dynakube.DynaKube) bool {
	return dk.Extensions().IsEnabled() && dk.Spec.TrustedCAs != "" || dk.TelemetryIngest().IsEnabled() && dk.IsCACertificateNeeded()
}
