package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	// Volume names and paths
	caCertsVolumeName = "cacerts"

	trustedCAsFile = "rootca.pem"
)

func setVolumes(dk *dynakube.DynaKube) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.Volumes = []corev1.Volume{
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
		}
		if dk.Spec.TrustedCAs != "" {
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
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

		o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: dk.ExtensionsTLSSecretName(),
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
		})
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
		Name:      dk.ExtensionsTLSSecretName(),
		MountPath: customEecTLSCertificatePath,
		ReadOnly:  true,
	})

	return vm
}
