// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package modifiers

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8ssecuritycontext"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = KubernetesMonitoringModifier{}
var _ volumeMountModifier = KubernetesMonitoringModifier{}
var _ initContainerModifier = KubernetesMonitoringModifier{}
var _ builder.Modifier = KubernetesMonitoringModifier{}

func NewKubernetesMonitoringModifier(dk dynakube.DynaKube, capability capability.Capability) KubernetesMonitoringModifier {
	return KubernetesMonitoringModifier{
		dk:         dk,
		capability: capability,
	}
}

type KubernetesMonitoringModifier struct {
	capability capability.Capability
	dk         dynakube.DynaKube
}

func (mod KubernetesMonitoringModifier) Enabled() bool {
	return mod.dk.ActiveGate().IsKubernetesMonitoringEnabled()
}

func (mod KubernetesMonitoringModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := k8scontainer.FindInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
	sts.Spec.Template.Spec.InitContainers = append(sts.Spec.Template.Spec.InitContainers, mod.getInitContainers()...)
	sts.Spec.Template.Spec.AutomountServiceAccountToken = new(true)

	return nil
}

func (mod KubernetesMonitoringModifier) getInitContainers() []corev1.Container {
	volumeMounts := slices.Concat([]corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      consts.TrustStoreVolumeName,
			MountPath: consts.GatewaySslMountPath,
		},
	}, mod.getReadOnlyInitVolumeMounts())

	securityContext := GetSecurityContext(true)
	securityContext.AppArmorProfile = k8ssecuritycontext.GetAppArmorProfile(mod.dk.ActiveGate().Annotations, consts.InitContainerName)

	return []corev1.Container{
		{
			Name:            consts.InitContainerName,
			Image:           mod.dk.ActiveGate().GetImage(),
			ImagePullPolicy: mod.dk.ActiveGate().ImagePullPolicy,
			WorkingDir:      consts.InitCertLoaderWorkDirMountPath,
			Command:         []string{"/bin/bash"},
			Args:            []string{"-c", consts.K8scrt2jksPath},
			VolumeMounts:    volumeMounts,
			Resources:       mod.capability.Properties().Resources,
			SecurityContext: securityContext,
		},
	}
}

func (mod KubernetesMonitoringModifier) getVolumes() []corev1.Volume {
	return slices.Concat([]corev1.Volume{
		{
			Name: consts.TrustStoreVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}, mod.getReadOnlyInitVolumes())
}

func (mod KubernetesMonitoringModifier) getReadOnlyInitVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name:         consts.InitCertLoaderWorkDirVolumeName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
	}
}

func (mod KubernetesMonitoringModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      consts.TrustStoreVolumeName,
			MountPath: consts.TrustStoreCacertsMountPath,
			SubPath:   consts.K8sCertificateFile,
		},
	}
}

func (mod KubernetesMonitoringModifier) getReadOnlyInitVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      consts.InitCertLoaderWorkDirVolumeName,
			MountPath: consts.InitCertLoaderWorkDirMountPath,
		},
	}
}

func GetSecurityContext(readOnlyRootFileSystem bool) *corev1.SecurityContext {
	securityContext := corev1.SecurityContext{
		Privileged:               new(false),
		AllowPrivilegeEscalation: new(false),
		RunAsNonRoot:             new(true),
		RunAsUser:                new(consts.DockerImageUser),
		RunAsGroup:               new(consts.DockerImageGroup),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		ReadOnlyRootFilesystem: new(readOnlyRootFileSystem),
	}

	return &securityContext
}
