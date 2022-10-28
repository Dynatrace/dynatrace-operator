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
	envMaxHeap         = "DT_MAX_HEAP_MEMORY"
	envMaxHeapDefault  = "1024m"
	envNodeType        = "DT_NODE_SIZE"
	envNodeTypeDefault = "S"

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

var livenessCmd = []string{
	"/bin/sh",
	"-c",
	"curl http://localhost:7878/command/version",
}

func newSyntheticModifier(kube dynatracev1beta1.DynaKube) SyntheticModifier {
	return SyntheticModifier{
		dynakube: kube,
	}
}

func (syn SyntheticModifier) Enabled() bool {
	return syn.dynakube.IsSyntheticActiveGateEnabled()
}

func (syn SyntheticModifier) Modify(sts *appsv1.StatefulSet) {
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
	sts.Spec.Template.Spec.SecurityContext.FSGroup = address.Of(int64(1001))

	baseContainer := kubeobjects.FindContainerInPodSpec(
		&sts.Spec.Template.Spec,
		consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(
		baseContainer.VolumeMounts,
		buildPublicVolumeMounts()...)
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

// make the compiler watch the implemented volumeMountModifier
var _ volumeMountModifier = (*SyntheticModifier)(nil)

func (syn SyntheticModifier) getVolumeMounts() []corev1.VolumeMount {
	private := []corev1.VolumeMount{
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
		private,
		buildPublicVolumeMounts()...)
}

var _ envModifier = (*SyntheticModifier)(nil)

func (syn SyntheticModifier) getEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: envMaxHeap, Value: envMaxHeapDefault},
		{Name: envNodeType, Value: envNodeTypeDefault},
	}
}

func (syn SyntheticModifier) buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               address.Of(false),
		AllowPrivilegeEscalation: address.Of(false),
		ReadOnlyRootFilesystem:   address.Of(true),
		RunAsNonRoot:             address.Of(true),
		RunAsUser:                address.Of(int64(1001)),

		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}
}

var _ dynatracev1beta1.ResourceRequirementer = (*SyntheticModifier)(nil)

func (syn SyntheticModifier) Limits(res corev1.ResourceName) *resource.Quantity {
	return syn.dynakube.FeatureSyntheticResourcesLimits(res)
}

func (syn SyntheticModifier) Requests(res corev1.ResourceName) *resource.Quantity {
	return syn.dynakube.FeatureSyntheticResourcesRequests(res)
}

func (syn SyntheticModifier) buildResources() corev1.ResourceRequirements {
	return dynatracev1beta1.BuildResourceRequirements(syn)
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

var _ volumeModifier = (*SyntheticModifier)(nil)

func (syn SyntheticModifier) getVolumes() []corev1.Volume {
	buildLimit := func(quantity string) *resource.Quantity {
		built := resource.MustParse(quantity)
		return &built
	}

	return []corev1.Volume{
		{
			Name: ChromiumCacheMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium:    "Memory",
					SizeLimit: buildLimit("512Mi"),
				},
			},
		},
		{
			Name: TmpStorageMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: buildLimit("10Mi"),
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
						corev1.ResourceStorage: resource.MustParse("6Gi"),
					},
				},
			},
		},
	}
}
