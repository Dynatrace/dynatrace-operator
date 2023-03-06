package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	_ "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	envNodeType   = "DT_NODE_SIZE"
	envMaxHeap    = "DT_MAX_HEAP_MEMORY"
	envLocationId = "DT_LOCATION_ID"

	ChromiumCacheMountName = "chromium-cache"
	chromiumCacheMountPath = "/var/tmp/dynatrace/synthetic/cache"

	ArchiveStorageMountName          = "archive-storage"
	synLogArchiveStorageMountPath    = "/var/log/dynatrace/synthetic"
	synLogArchiveStorageMountSubPath = "syn-log"
	synTmpArchiveStorageMountPath    = "/var/tmp/dynatrace/synthetic"
	synTmpArchiveStorageMountSubPath = "syn-tmp"

	TmpStorageMountName             = "tmp-storage"
	vucConfigTmpStorageMountPath    = "/var/lib/dynatrace/synthetic/config"
	vucConfigTmpStorageMountSubPath = "vuc-config"
	xvfbTmpStorageMountPath         = "/tmp"
	xvfbTmpStorageMountSubPath      = "xvfb-tmp"
	gatewayConfigMountSubPath       = "ag-config"
)

type SyntheticModifier struct {
	dynaKube dynatracev1beta1.DynaKube
}

// make the compiler watch the implemented interfaces
var (
	_ volumeMountModifier = (*SyntheticModifier)(nil)
	_ envModifier         = (*SyntheticModifier)(nil)
	_ volumeModifier      = (*SyntheticModifier)(nil)
)

type nodeRequirements struct {
	requestResources corev1.ResourceList
	limitResources   corev1.ResourceList

	jvmHeap              *resource.Quantity
	chromiumCacheVolume  *resource.Quantity
	tmpStorageVolume     *resource.Quantity
	supportArchiveVolume *resource.Quantity
}

var nodeRequirementsBySize = map[string]nodeRequirements{
	dynatracev1beta1.SyntheticNodeXs: {
		requestResources:     kubeobjects.NewResources("1", "2Gi"),
		limitResources:       kubeobjects.NewResources("2", "3Gi"),
		jvmHeap:              kubeobjects.NewQuantity("700M"),
		chromiumCacheVolume:  kubeobjects.NewQuantity("256Mi"),
		tmpStorageVolume:     kubeobjects.NewQuantity("8Mi"),
		supportArchiveVolume: kubeobjects.NewQuantity("3Gi"),
	},

	dynatracev1beta1.SyntheticNodeS: {
		requestResources:     kubeobjects.NewResources("2", "3Gi"),
		limitResources:       kubeobjects.NewResources("4", "6Gi"),
		jvmHeap:              kubeobjects.NewQuantity("1024M"),
		chromiumCacheVolume:  kubeobjects.NewQuantity("512Mi"),
		tmpStorageVolume:     kubeobjects.NewQuantity("10Mi"),
		supportArchiveVolume: kubeobjects.NewQuantity("6Gi"),
	},

	dynatracev1beta1.SyntheticNodeM: {
		requestResources:     kubeobjects.NewResources("4", "5Gi"),
		limitResources:       kubeobjects.NewResources("8", "10Gi"),
		jvmHeap:              kubeobjects.NewQuantity("2048M"),
		chromiumCacheVolume:  kubeobjects.NewQuantity("1Gi"),
		tmpStorageVolume:     kubeobjects.NewQuantity("12Mi"),
		supportArchiveVolume: kubeobjects.NewQuantity("12Gi"),
	},
}

func (syn SyntheticModifier) nodeRequirements() nodeRequirements {
	return nodeRequirementsBySize[syn.dynaKube.FeatureSyntheticNodeType()]
}

var (
	livenessCmd = []string{
		"/bin/sh",
		"-c",
		"curl http://localhost:7878/command/version",
	}
	ActiveGateResourceRequirements = corev1.ResourceRequirements{
		Limits:   kubeobjects.NewResources("300m", "1Gi"),
		Requests: kubeobjects.NewResources("150m", "250Mi"),
	}
)

func newSyntheticModifier(dynaKube dynatracev1beta1.DynaKube) SyntheticModifier {
	return SyntheticModifier{
		dynaKube: dynaKube,
	}
}

func (syn SyntheticModifier) Enabled() bool {
	return syn.dynaKube.IsSyntheticMonitoringEnabled()
}

func (syn SyntheticModifier) Modify(sts *appsv1.StatefulSet) error {
	sts.Spec.Template.Spec.Containers = append(
		sts.Spec.Template.Spec.Containers,
		syn.buildContainer())
	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		syn.getVolumes()...)
	sts.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}
	sts.Spec.Template.Spec.SecurityContext.FSGroup = address.Of[int64](1001)

	baseContainer := kubeobjects.FindContainerInPodSpec(
		&sts.Spec.Template.Spec,
		consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(
		baseContainer.VolumeMounts,
		buildPublicVolumeMounts()...)
	baseContainer.Env = append(
		baseContainer.Env,
		corev1.EnvVar{
			Name:  envLocationId,
			Value: syn.dynaKube.FeatureSyntheticLocationEntityId(),
		})

	return nil
}

func (syn SyntheticModifier) buildContainer() corev1.Container {
	container := corev1.Container{
		Name:            consts.SyntheticContainerName,
		Image:           syn.image(),
		ImagePullPolicy: corev1.PullAlways,
		Env:             syn.getEnvs(),
		VolumeMounts:    syn.getVolumeMounts(),
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: livenessCmd,
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       30,
			FailureThreshold:    2,
			SuccessThreshold:    1,
			TimeoutSeconds:      3,
		},
		SecurityContext: syn.buildSecurityContext(),
		Resources:       syn.buildResources(),
	}
	return container
}

func (syn SyntheticModifier) image() string {
	return syn.dynaKube.SyntheticImage()
}

func (syn SyntheticModifier) getVolumeMounts() []corev1.VolumeMount {
	privateMounts := []corev1.VolumeMount{
		{
			Name:      ChromiumCacheMountName,
			MountPath: chromiumCacheMountPath,
		},
		{
			Name:      TmpStorageMountName,
			SubPath:   xvfbTmpStorageMountSubPath,
			MountPath: xvfbTmpStorageMountPath,
		},
	}
	return append(
		privateMounts,
		buildPublicVolumeMounts()...)
}

func (syn SyntheticModifier) getEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  envNodeType,
			Value: syn.dynaKube.FeatureSyntheticNodeType(),
		},
		{
			Name:  envMaxHeap,
			Value: syn.nodeRequirements().jvmHeap.String(),
		},
	}
}

func (syn SyntheticModifier) buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               address.Of(false),
		AllowPrivilegeEscalation: address.Of(false),
		ReadOnlyRootFilesystem:   address.Of(true),
		RunAsNonRoot:             address.Of(true),
		RunAsUser:                address.Of[int64](1001),

		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}
}

func (syn SyntheticModifier) buildResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits:   syn.nodeRequirements().limitResources,
		Requests: syn.nodeRequirements().requestResources,
	}
}

func buildPublicVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      ArchiveStorageMountName,
			SubPath:   synLogArchiveStorageMountSubPath,
			MountPath: synLogArchiveStorageMountPath,
		},
		{
			Name:      ArchiveStorageMountName,
			SubPath:   synTmpArchiveStorageMountSubPath,
			MountPath: synTmpArchiveStorageMountPath,
		},
		{
			Name:      TmpStorageMountName,
			SubPath:   vucConfigTmpStorageMountSubPath,
			MountPath: vucConfigTmpStorageMountPath,
		},
		{
			Name:      TmpStorageMountName,
			SubPath:   gatewayConfigMountSubPath,
			MountPath: consts.GatewayConfigMountPoint,
		},
	}
}

func (syn SyntheticModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: ChromiumCacheMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium:    "Memory",
					SizeLimit: syn.nodeRequirements().chromiumCacheVolume,
				},
			},
		},
		{
			Name: TmpStorageMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: syn.nodeRequirements().tmpStorageVolume,
				},
			},
		},
		{
			Name: ArchiveStorageMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: syn.nodeRequirements().supportArchiveVolume,
				},
			},
		},
	}
}
