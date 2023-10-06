package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	_ "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	kubeobjects2 "github.com/Dynatrace/dynatrace-operator/src/util/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/util/kubeobjects/address"
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
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
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
		requestResources:     kubeobjects2.NewResources("1", "2Gi"),
		limitResources:       kubeobjects2.NewResources("2", "3Gi"),
		jvmHeap:              kubeobjects2.NewQuantity("700M"),
		chromiumCacheVolume:  kubeobjects2.NewQuantity("256Mi"),
		tmpStorageVolume:     kubeobjects2.NewQuantity("8Mi"),
		supportArchiveVolume: kubeobjects2.NewQuantity("3Gi"),
	},

	dynatracev1beta1.SyntheticNodeS: {
		requestResources:     kubeobjects2.NewResources("2", "3Gi"),
		limitResources:       kubeobjects2.NewResources("4", "6Gi"),
		jvmHeap:              kubeobjects2.NewQuantity("1024M"),
		chromiumCacheVolume:  kubeobjects2.NewQuantity("512Mi"),
		tmpStorageVolume:     kubeobjects2.NewQuantity("10Mi"),
		supportArchiveVolume: kubeobjects2.NewQuantity("6Gi"),
	},

	dynatracev1beta1.SyntheticNodeM: {
		requestResources:     kubeobjects2.NewResources("4", "5Gi"),
		limitResources:       kubeobjects2.NewResources("8", "10Gi"),
		jvmHeap:              kubeobjects2.NewQuantity("2048M"),
		chromiumCacheVolume:  kubeobjects2.NewQuantity("1Gi"),
		tmpStorageVolume:     kubeobjects2.NewQuantity("12Mi"),
		supportArchiveVolume: kubeobjects2.NewQuantity("12Gi"),
	},
}

func (modifier SyntheticModifier) nodeRequirements() nodeRequirements {
	return nodeRequirementsBySize[modifier.dynakube.FeatureSyntheticNodeType()]
}

var (
	livenessCmd = []string{
		"/bin/sh",
		"-c",
		"curl http://localhost:7878/command/version",
	}
)

func newSyntheticModifier(
	dynakube dynatracev1beta1.DynaKube,
	capability capability.Capability,
) SyntheticModifier {
	return SyntheticModifier{
		dynakube:   dynakube,
		capability: capability,
	}
}

func (modifier SyntheticModifier) Enabled() bool {
	return modifier.dynakube.IsSyntheticMonitoringEnabled()
}

func (modifier SyntheticModifier) Modify(sts *appsv1.StatefulSet) error {
	version := modifier.dynakube.Status.Synthetic.Version
	sts.Labels[kubeobjects2.AppVersionLabel] = version
	sts.Labels[kubeobjects2.AppComponentLabel] = kubeobjects2.SyntheticComponentLabel

	sts.Spec.Template.Spec.Containers = append(
		sts.Spec.Template.Spec.Containers,
		modifier.buildContainer())
	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		modifier.getVolumes()...)
	sts.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}
	sts.Spec.Template.Spec.SecurityContext.FSGroup = address.Of[int64](1001)

	baseContainer := kubeobjects2.FindContainerInPodSpec(
		&sts.Spec.Template.Spec,
		consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(
		baseContainer.VolumeMounts,
		buildPublicVolumeMounts()...)
	baseContainer.Env = append(
		baseContainer.Env,
		corev1.EnvVar{
			Name:  envLocationId,
			Value: modifier.dynakube.FeatureSyntheticLocationEntityId(),
		})

	return nil
}

func (modifier SyntheticModifier) buildContainer() corev1.Container {
	container := corev1.Container{
		Name:            consts.SyntheticContainerName,
		Image:           modifier.image(),
		ImagePullPolicy: corev1.PullAlways,
		Env:             modifier.getEnvs(),
		VolumeMounts:    modifier.getVolumeMounts(),
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
		SecurityContext: modifier.buildSecurityContext(),
		Resources:       modifier.buildResources(),
	}
	return container
}

func (modifier SyntheticModifier) image() string {
	return modifier.dynakube.SyntheticImage()
}

func (modifier SyntheticModifier) getVolumeMounts() []corev1.VolumeMount {
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

func (modifier SyntheticModifier) getEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  envNodeType,
			Value: modifier.dynakube.FeatureSyntheticNodeType(),
		},
		{
			Name:  envMaxHeap,
			Value: modifier.nodeRequirements().jvmHeap.String(),
		},
	}
}

func (modifier SyntheticModifier) buildSecurityContext() *corev1.SecurityContext {
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

func (modifier SyntheticModifier) buildResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits:   modifier.nodeRequirements().limitResources,
		Requests: modifier.nodeRequirements().requestResources,
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

func (modifier SyntheticModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: ChromiumCacheMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium:    "Memory",
					SizeLimit: modifier.nodeRequirements().chromiumCacheVolume,
				},
			},
		},
		{
			Name: TmpStorageMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: modifier.nodeRequirements().tmpStorageVolume,
				},
			},
		},
		{
			Name: ArchiveStorageMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: modifier.nodeRequirements().supportArchiveVolume,
				},
			},
		},
	}
}
