package modifiers

import (
	"path/filepath"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = CertificatesModifier{}
var _ volumeMountModifier = CertificatesModifier{}
var _ builder.Modifier = CertificatesModifier{}

const (
	jettyCerts     = "server-certs"
	secretsRootDir = "/var/lib/dynatrace/secrets/"
)

func NewCertificatesModifier(dynakube dynatracev1beta2.DynaKube) CertificatesModifier {
	return CertificatesModifier{
		dynakube: dynakube,
	}
}

type CertificatesModifier struct {
	dynakube dynatracev1beta2.DynaKube
}

func (mod CertificatesModifier) Enabled() bool {
	return mod.dynakube.HasActiveGateCaCert()
}

func (mod CertificatesModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := container.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod CertificatesModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: jettyCerts,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: mod.dynakube.Spec.ActiveGate.TlsSecretName,
				},
			},
		},
	}
}

func (mod CertificatesModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      jettyCerts,
			MountPath: filepath.Join(secretsRootDir, "tls"),
		},
	}
}
