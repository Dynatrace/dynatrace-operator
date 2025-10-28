package volumes

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/mounts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

var (
	log = logd.Get().WithName("volumes-mutation")
)

const (
	ConfigVolumeName             = "dynatrace-config"
	InitConfigMountPath          = "/mnt/config"
	ConfigMountPath              = "/var/lib/dynatrace"
	ConfigMountSubPathOneAgent   = "/oneagent"
	ConfigMountPathOneAgent      = ConfigMountPath + ConfigMountSubPathOneAgent
	ConfigMountSubPathEnrichment = "/enrichment"
	ConfigMountPathEnrichment    = ConfigMountPath + ConfigMountSubPathEnrichment

	InputVolumeName    = "dynatrace-input"
	InitInputMountPath = "/mnt/input"

	// AnnotationResourcePrefix is used as a prefix for all volume resource annotations.
	AnnotationResourcePrefix = "volume.dynatrace.com/"

	// AnnotationConfigVolumeNameResource is used to specify the volume size for EmptyDir for dynatrace-config.
	AnnotationConfigVolumeNameResource = AnnotationResourcePrefix + ConfigVolumeName
)

func AddConfigVolume(pod *corev1.Pod) {
	if volumes.IsIn(pod.Spec.Volumes, ConfigVolumeName) {
		return
	}

	emptyDirVS := corev1.EmptyDirVolumeSource{}

	if r, ok := pod.Annotations[AnnotationConfigVolumeNameResource]; ok && r != "" {
		sizeLimit, err := resource.ParseQuantity(r)
		if err != nil {
			log.Error(err, "failed to parse quantity from annotation "+AnnotationConfigVolumeNameResource, "value", r)
		} else {
			emptyDirVS = corev1.EmptyDirVolumeSource{
				SizeLimit: &sizeLimit,
			}
		}
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: ConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &emptyDirVS,
			},
		},
	)
}

func IsConfigVolumeMountNeeded(container *corev1.Container, splitMountsEnabled bool) bool {
	if !splitMountsEnabled && !mounts.IsPathIn(container.VolumeMounts, ConfigMountPath) {
		return true
	}

	return false
}

func AddConfigVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      ConfigVolumeName,
			MountPath: ConfigMountPath,
			SubPath:   container.Name,
		},
	)
}

func IsOneAgentConfigVolumeMountNeeded(container *corev1.Container, splitMountsEnabled bool, isAppInjectionEnabled bool) bool {
	if splitMountsEnabled && !mounts.IsPathIn(container.VolumeMounts, ConfigMountPathOneAgent) && isAppInjectionEnabled {
		return true
	}

	return false
}

func AddOneAgentConfigVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      ConfigVolumeName,
			MountPath: ConfigMountPathOneAgent,
			SubPath:   container.Name + ConfigMountSubPathOneAgent,
		},
	)
}

func IsEnrichmentConfigVolumeMountNeeded(container *corev1.Container, splitMountsEnabled bool, isMetadataEnrichmentEnabled bool) bool {
	if splitMountsEnabled && !mounts.IsPathIn(container.VolumeMounts, ConfigMountPathEnrichment) && isMetadataEnrichmentEnabled {
		return true
	}

	return false
}

func AddEnrichmentConfigVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      ConfigVolumeName,
			MountPath: ConfigMountPathEnrichment,
			SubPath:   container.Name + ConfigMountSubPathEnrichment,
		},
	)
}

func AddInitConfigVolumeMount(container *corev1.Container) {
	if mounts.IsPathIn(container.VolumeMounts, InitConfigMountPath) {
		return
	}

	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      ConfigVolumeName,
			MountPath: InitConfigMountPath,
		},
	)
}

func AddInputVolume(pod *corev1.Pod) {
	if volumes.IsIn(pod.Spec.Volumes, InputVolumeName) {
		return
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: InputVolumeName,
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: consts.BootstrapperInitSecretName,
								},
								Optional: ptr.To(false),
							},
						},
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: consts.BootstrapperInitCertsSecretName,
								},
								Optional: ptr.To(true),
							},
						},
					},
				},
			},
		},
	)
}

func AddInitInputVolumeMount(container *corev1.Container) {
	if mounts.IsPathIn(container.VolumeMounts, InitInputMountPath) {
		return
	}

	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      InputVolumeName,
			MountPath: InitInputMountPath,
			ReadOnly:  true,
		},
	)
}
