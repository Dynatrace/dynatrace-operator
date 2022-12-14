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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	envNodeType = "DT_NODE_SIZE"
	envMaxHeap  = "DT_MAX_HEAP_MEMORY"

	ChromiumCacheMountName = "chromium-cache"
	chromiumCacheMountPath = "/var/tmp/dynatrace/synthetic/cache"

	PersistentStorageMountName          = "persistent-storage"
	synLogPersistentStorageMountPath    = "/var/log/dynatrace/synthetic"
	synLogPersistentStorageMountSubPath = "syn-log"
	synTmpPersistentStorageMountPath    = "/var/tmp/dynatrace/synthetic"
	synTmpPersistentStorageMountSubPath = "syn-tmp"

	TmpStorageMountName             = "tmp-storage"
	vucConfigTmpStorageMountPath    = "/var/lib/dynatrace/synthetic/config"
	vucConfigTmpStorageMountSubPath = "vuc-config"
	xvfbTmpStorageMountPath         = "/tmp"
	xvfbTmpStorageMountSubPath      = "xvfb-tmp"
)

type SyntheticModifier struct {
	dynakube dynatracev1beta1.DynaKube
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

	jvmHeap                              *resource.Quantity
	chromiumCacheVolume                  *resource.Quantity
	tmpStorageVolume                     *resource.Quantity
	persistentVolumeClaimResourceStorage resource.Quantity
}

var nodeRequirementsBySize = map[string]nodeRequirements{
	dynatracev1beta1.SyntheticNodeXs: {
		requestResources:                     buildRequirementResources("1", "2Gi"),
		limitResources:                       buildRequirementResources("2", "3Gi"),
		jvmHeap:                              buildQuantity("700M"),
		chromiumCacheVolume:                  buildQuantity("256Mi"),
		tmpStorageVolume:                     buildQuantity("8Mi"),
		persistentVolumeClaimResourceStorage: *buildQuantity("3Gi"),
	},

	dynatracev1beta1.SyntheticNodeS: {
		requestResources:                     buildRequirementResources("2", "3Gi"),
		limitResources:                       buildRequirementResources("4", "6Gi"),
		jvmHeap:                              buildQuantity("1024M"),
		chromiumCacheVolume:                  buildQuantity("512Mi"),
		tmpStorageVolume:                     buildQuantity("10Mi"),
		persistentVolumeClaimResourceStorage: *buildQuantity("6Gi"),
	},

	dynatracev1beta1.SyntheticNodeM: {
		requestResources:                     buildRequirementResources("4", "5Gi"),
		limitResources:                       buildRequirementResources("8", "10Gi"),
		jvmHeap:                              buildQuantity("2048M"),
		chromiumCacheVolume:                  buildQuantity("1Gi"),
		tmpStorageVolume:                     buildQuantity("12Mi"),
		persistentVolumeClaimResourceStorage: *buildQuantity("12Gi"),
	},
}

func (syn SyntheticModifier) nodeRequirements() nodeRequirements {
	return nodeRequirementsBySize[syn.dynakube.FeatureSyntheticNodeType()]
}

var (
	livenessCmd = []string{
		"/bin/sh",
		"-c",
		"curl http://localhost:7878/command/version",
	}
	ActiveGateResourceRequirements = corev1.ResourceRequirements{
		Limits:   buildRequirementResources("300m", "1Gi"),
		Requests: buildRequirementResources("150m", "250Mi"),
	}
)

func newSyntheticModifier(dynakube dynatracev1beta1.DynaKube) SyntheticModifier {
	return SyntheticModifier{
		dynakube: dynakube,
	}
}

func (syn SyntheticModifier) Enabled() bool {
	return syn.dynakube.IsSyntheticActiveGateEnabled()
}

func (syn SyntheticModifier) Modify(sts *appsv1.StatefulSet) error {
	sts.Spec.Template.Spec.Containers = append(
		sts.Spec.Template.Spec.Containers,
		syn.buildContainer())
	sts.Spec.VolumeClaimTemplates = append(
		sts.Spec.VolumeClaimTemplates,
		syn.buildVolumeClaimTemplates()...)
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
	return syn.dynakube.SyntheticImage()
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
			Value: syn.dynakube.FeatureSyntheticNodeType(),
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
			Name:      PersistentStorageMountName,
			SubPath:   synLogPersistentStorageMountSubPath,
			MountPath: synLogPersistentStorageMountPath,
		},
		{
			Name:      PersistentStorageMountName,
			SubPath:   synTmpPersistentStorageMountSubPath,
			MountPath: synTmpPersistentStorageMountPath,
		},
		{
			Name:      TmpStorageMountName,
			SubPath:   vucConfigTmpStorageMountSubPath,
			MountPath: vucConfigTmpStorageMountPath,
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
	}
}

func (syn SyntheticModifier) buildVolumeClaimTemplates() []corev1.PersistentVolumeClaim {
	return []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: PersistentStorageMountName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: syn.nodeRequirements().persistentVolumeClaimResourceStorage,
					},
				},
			},
		},
	}
}

func buildRequirementResources(cpu, memory string) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:    *buildQuantity(cpu),
		corev1.ResourceMemory: *buildQuantity(memory),
	}
}

func buildQuantity(serialized string) *resource.Quantity {
	built := resource.MustParse(serialized)
	return &built
}
