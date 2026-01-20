package oneagent

import (
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/preload"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes/app"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8svolume"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

const (
	BinVolumeName    = "oneagent-bin"
	ldPreloadPath    = "/etc/ld.so.preload"
	ldPreloadSubPath = preload.ConfigPath
)

func addVolumeMounts(container *corev1.Container, installPath string) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      BinVolumeName,
			MountPath: installPath,
			ReadOnly:  true,
		},
		corev1.VolumeMount{
			Name:      volumes.ConfigVolumeName,
			MountPath: ldPreloadPath,
			SubPath:   ldPreloadSubPath,
		},
	)
}

func addInitBinMount(initContainer *corev1.Container, readonly bool) {
	initContainer.VolumeMounts = append(initContainer.VolumeMounts,
		corev1.VolumeMount{
			Name:      BinVolumeName,
			MountPath: consts.AgentInitBinDirMount,
			ReadOnly:  readonly,
		},
	)
}

func addEmptyDirBinVolume(pod *corev1.Pod) {
	if k8svolume.Contains(pod.Spec.Volumes, BinVolumeName) {
		return
	}

	emptyDirVS := corev1.EmptyDirVolumeSource{}

	if r, ok := pod.Annotations[AnnotationOneAgentBinResource]; ok && r != "" {
		sizeLimit, err := resource.ParseQuantity(r)
		if err != nil {
			log.Error(err, "failed to parse quantity from annotation "+AnnotationOneAgentBinResource, "value", r)
		} else {
			emptyDirVS = corev1.EmptyDirVolumeSource{
				SizeLimit: &sizeLimit,
			}
		}
	}

	volumeSource := corev1.VolumeSource{
		EmptyDir: &emptyDirVS,
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name:         BinVolumeName,
			VolumeSource: volumeSource,
		},
	)
}

func addCSIBinVolume(pod *corev1.Pod, dkName string, maxTimeout string) {
	if k8svolume.Contains(pod.Spec.Volumes, BinVolumeName) {
		return
	}

	volumeSource := corev1.VolumeSource{
		CSI: &corev1.CSIVolumeSource{
			Driver:   dtcsi.DriverName,
			ReadOnly: ptr.To(true),
			VolumeAttributes: map[string]string{
				csivolumes.CSIVolumeAttributeModeField:     appvolumes.Mode,
				csivolumes.CSIVolumeAttributeDynakubeField: dkName,
				csivolumes.CSIVolumeAttributeRetryTimeout:  maxTimeout,
			},
		},
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name:         BinVolumeName,
			VolumeSource: volumeSource,
		},
	)
}
