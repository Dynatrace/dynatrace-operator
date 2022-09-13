package modifiers

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	statsdProbesPortName = "statsd-probes"
	statsdProbesPort     = 14999
	statsdLogsDir        = extensionsLogsDir + "/datasources-statsd"

	dataSourceMetadata   = "ds-metadata"
	dataSourceStatsdLogs = "statsd-logs"

	envStatsdMetadata            = "StatsdMetadataDir"
	envDataSourceProbeServerPort = "ProbeServerPort"
	envDataSourceLogFile         = "DsLogFile"
	envStatsdStartupArgsPath     = "StatsdExecArgsPath"

	dataSourceStartupArgsMountPoint = "/mnt/dsexecargs"
	dataSourceAuthTokenMountPoint   = "/var/lib/dynatrace/remotepluginmodule/agent/runtime/datasources"
	dataSourceMetadataMountPoint    = "/mnt/dsmetadata"
	statsdMetadataMountPoint        = "/opt/dynatrace/remotepluginmodule/agent/datasources/statsd"
)

var _ builder.Modifier = StatsdModifier{}

type StatsdModifier struct {
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
}

func NewStatsdModifier(dynakube dynatracev1beta1.DynaKube, capability capability.Capability) StatsdModifier {
	return StatsdModifier{
		dynakube:   dynakube,
		capability: capability,
	}
}

func (statsd StatsdModifier) Enabled() bool {
	return statsd.dynakube.IsStatsdCapabilityEnabled()
}

func (statsd StatsdModifier) Modify(sts *appsv1.StatefulSet) {
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, statsd.buildContainer())
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, statsd.getVolumes(sts.Spec.Template.Spec.Volumes)...)

	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, statsd.getActiveGateVolumeMounts(baseContainer.VolumeMounts)...)

}

func (statsd *StatsdModifier) getActiveGateVolumeMounts(presentMounts []corev1.VolumeMount) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{{Name: dataSourceStatsdLogs, MountPath: extensionsLogsDir + "/statsd", ReadOnly: true}}
	neededMount := corev1.VolumeMount{
		ReadOnly:  false,
		Name:      consts.GatewayConfigVolumeName,
		MountPath: consts.GatewayConfigMountPoint,
	}
	if !kubeobjects.IsVolumeMountPresent(presentMounts, neededMount) {
		volumeMounts = append(volumeMounts, neededMount)
	}
	return volumeMounts
}

func (statsd StatsdModifier) getVolumes(presentVolumes []corev1.Volume) []corev1.Volume {
	volumes := statsd.buildVolumes()
	_, err := kubeobjects.GetVolumeByName(presentVolumes, consts.GatewayConfigVolumeName)
	if err != nil {
		volumes = append(volumes,
			corev1.Volume{
				Name: consts.GatewayConfigVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
	}
	return volumes
}

func (statsd StatsdModifier) buildContainer() corev1.Container {
	container := corev1.Container{
		Name:            consts.StatsdContainerName,
		Image:           statsd.image(),
		ImagePullPolicy: corev1.PullAlways,
		Env:             statsd.buildEnvs(),
		VolumeMounts:    statsd.buildVolumeMounts(),
		Command:         statsd.buildCommand(),
		Ports:           statsd.buildPorts(),
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.IntOrString{IntVal: statsdProbesPort},
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       15,
			FailureThreshold:    3,
			SuccessThreshold:    1,
			TimeoutSeconds:      1,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/livez",
					Port: intstr.IntOrString{IntVal: statsdProbesPort},
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       15,
			FailureThreshold:    3,
			SuccessThreshold:    1,
			TimeoutSeconds:      1,
		},
		SecurityContext: statsd.buildSecurityContext(),
		Resources:       statsd.buildResourceRequirements(),
	}
	if statsd.dynakube.NeedsActiveGateServicePorts() {
		container.Ports = []corev1.ContainerPort{
			{
				Name:          consts.StatsdIngestTargetPort,
				ContainerPort: consts.StatsdIngestPort,
				Protocol:      corev1.ProtocolUDP,
			},
		}
	}
	return container
}

func (statsd StatsdModifier) buildVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: dataSourceMetadata,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: dataSourceStatsdLogs,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func (statsd StatsdModifier) image() string {
	if statsd.dynakube.FeatureUseActiveGateImageForStatsd() {
		return statsd.dynakube.ActiveGateImage()
	}
	return statsd.dynakube.StatsdImage()
}

func (statsd StatsdModifier) buildCommand() []string {
	if statsd.dynakube.FeatureUseActiveGateImageForStatsd() {
		return []string{
			"/bin/bash", "/dt/statsd/entrypoint.sh",
		}
	}
	return nil
}

func (statsd StatsdModifier) buildPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{Name: consts.StatsdIngestTargetPort, ContainerPort: consts.StatsdIngestPort},
		{Name: statsdProbesPortName, ContainerPort: statsdProbesPort},
	}
}

func (statsd StatsdModifier) buildVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: dataSourceStartupArguments, MountPath: dataSourceStartupArgsMountPoint},
		{Name: dataSourceAuthToken, MountPath: dataSourceAuthTokenMountPoint},
		{Name: dataSourceMetadata, MountPath: dataSourceMetadataMountPoint},
		{Name: dataSourceStatsdLogs, MountPath: statsdLogsDir},
	}
}

func (statsd StatsdModifier) buildEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: envStatsdStartupArgsPath, Value: dataSourceStartupArgsMountPoint + "/statsd.process.json"},
		{Name: envDataSourceProbeServerPort, Value: fmt.Sprintf("%d", statsdProbesPort)},
		{Name: envStatsdMetadata, Value: dataSourceMetadataMountPoint},
		{Name: envDataSourceLogFile, Value: statsdLogsDir + "/dynatracesourcestatsd.log"},
	}
}

func (statsd StatsdModifier) buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               address.Of(false),
		AllowPrivilegeEscalation: address.Of(false),
		ReadOnlyRootFilesystem:   address.Of(true),

		RunAsNonRoot: address.Of(true),
		RunAsUser:    address.Of(kubeobjects.UnprivilegedUser),
		RunAsGroup:   address.Of(kubeobjects.UnprivilegedGroup),

		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}
}

var _ dynatracev1beta1.ResourceRequirementer = (*StatsdModifier)(nil)

func (statsd StatsdModifier) Limits(resourceName corev1.ResourceName) *resource.Quantity {
	return statsd.dynakube.FeatureStatsdResourcesLimits(resourceName)
}

func (statsd StatsdModifier) Requests(resourceName corev1.ResourceName) *resource.Quantity {
	return statsd.dynakube.FeatureStatsdResourcesRequests(resourceName)
}

func (statsd StatsdModifier) buildResourceRequirements() corev1.ResourceRequirements {
	return dynatracev1beta1.BuildResourceRequirements(statsd)
}
