package volumes

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8smount"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8svolume"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	ConfigVolumeName    = "dynatrace-config"
	InitConfigMountPath = "/mnt/config"
	ConfigMountPath     = "/var/lib/dynatrace"

	InputVolumeName    = "dynatrace-input"
	InitInputMountPath = "/mnt/input"

	// AnnotationResourcePrefix is used as a prefix for all volume resource annotations.
	AnnotationResourcePrefix = "volume.dynatrace.com/"

	// AnnotationConfigVolumeNameResource is used to specify the volume size for EmptyDir for dynatrace-config.
	AnnotationConfigVolumeNameResource = AnnotationResourcePrefix + ConfigVolumeName

	// annotationDynatraceInjected is set to "true" by the webhook to Pods to indicate that it has been injected.
	annotationInjected = AnnotationResourcePrefix + "injected"
	// annotationDynatraceReason is add to provide extra info why an injection didn't happen.
	annotationReason = AnnotationResourcePrefix + "reason"
	// reasonConflictingVolumeType indicates that the user provided a volume definition that conflicts with the one that would be added by the mutator.
	reasonConflictingVolumeType = "ConflictingVolumeType"
)

// existingVolumeError indicates that an existing volume cannot be used for injection because it does not match the expected spec.
type existingVolumeError struct {
	volumeName string
}

func (e existingVolumeError) Error() string {
	return "user-provided " + e.volumeName + " volume cannot be mounted due to invalid configuration"
}

func AddConfigVolume(ctx context.Context, pod *corev1.Pod) error {
	if vol := k8svolume.FindByName(pod.Spec.Volumes, ConfigVolumeName); vol != nil {
		if vol.EmptyDir == nil {
			return dtwebhook.MutatorError{
				Err:      existingVolumeError{ConfigVolumeName},
				Annotate: setNotInjectedReason(reasonConflictingVolumeType),
			}
		}

		return nil
	}

	emptyDirVS := corev1.EmptyDirVolumeSource{}

	if r, ok := pod.Annotations[AnnotationConfigVolumeNameResource]; ok && r != "" {
		sizeLimit, err := resource.ParseQuantity(r)
		if err != nil {
			_, log := logd.NewFromContext(ctx, "volumes-mutation")
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

	return nil
}

func AddConfigVolumeMount(container *corev1.Container, request *dtwebhook.BaseRequest) {
	if request.IsSplitMountsEnabled() {
		addSplitMounts(container, request)
	} else {
		addCommonConfigVolumeMount(container)
	}
}

func addCommonConfigVolumeMount(container *corev1.Container) {
	vm := corev1.VolumeMount{
		Name:      ConfigVolumeName,
		MountPath: ConfigMountPath,
		SubPath:   container.Name,
	}
	container.VolumeMounts = k8smount.Append(container.VolumeMounts, vm)
}

func HasCommonConfigVolumeMounts(container *corev1.Container) bool {
	return k8smount.ContainsPath(container.VolumeMounts, ConfigMountPath)
}

func AddInitConfigVolumeMount(container *corev1.Container) {
	if k8smount.ContainsPath(container.VolumeMounts, InitConfigMountPath) {
		return
	}

	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      ConfigVolumeName,
			MountPath: InitConfigMountPath,
		},
	)
}

func AddInputVolume(pod *corev1.Pod) error {
	if vol := k8svolume.FindByName(pod.Spec.Volumes, InputVolumeName); vol != nil {
		if vol.Projected == nil || len(vol.Projected.Sources) != 2 ||
			vol.Projected.Sources[0].Secret == nil || vol.Projected.Sources[0].Secret.Name != consts.BootstrapperInitSecretName ||
			vol.Projected.Sources[1].Secret == nil || vol.Projected.Sources[1].Secret.Name != consts.BootstrapperInitCertsSecretName {
			return dtwebhook.MutatorError{
				Err:      existingVolumeError{InputVolumeName},
				Annotate: setNotInjectedReason(reasonConflictingVolumeType),
			}
		}

		return nil
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
								Optional: new(false),
							},
						},
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: consts.BootstrapperInitCertsSecretName,
								},
								Optional: new(true),
							},
						},
					},
				},
			},
		},
	)

	return nil
}

func AddInitInputVolumeMount(container *corev1.Container) {
	if k8smount.ContainsPath(container.VolumeMounts, InitInputMountPath) {
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

func setNotInjectedReason(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[annotationInjected] = "false"
		pod.Annotations[annotationReason] = reason
	}
}
