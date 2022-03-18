package statefulset

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address_of"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const statsdProbesPortName = "statsd-probes"
const statsdProbesPort = 14999
const statsdLogsDir = extensionsLogsDir + "/datasources-statsd"

const (
	dataSourceMetadata   = "ds-metadata"
	dataSourceStatsdLogs = "statsd-logs"

	envStatsdMetadata            = "StatsdMetadataDir"
	envDataSourceProbeServerPort = "ProbeServerPort"
	envDataSourceLogFile         = "DsLogFile"
	envStatsdStartupArgsPath     = "StatsdExecArgsPath"
)

var _ kubeobjects.ContainerBuilder = (*Statsd)(nil)

type Statsd struct {
	stsProperties *statefulSetProperties
}

func NewStatsd(stsProperties *statefulSetProperties) *Statsd {
	return &Statsd{
		stsProperties: stsProperties,
	}
}

func (statsd *Statsd) BuildContainer() corev1.Container {
	return corev1.Container{
		Name:            capability.StatsdContainerName,
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
}

func (statsd *Statsd) BuildVolumes() []corev1.Volume {
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

func (statsd *Statsd) image() string {
	if statsd.stsProperties.FeatureUseActiveGateImageForStatsd() {
		return statsd.stsProperties.ActiveGateImage()
	}
	return statsd.stsProperties.StatsdImage()
}

func (statsd *Statsd) buildCommand() []string {
	if statsd.stsProperties.DynaKube.FeatureUseActiveGateImageForStatsd() {
		return []string{
			"/bin/bash", "/dt/statsd/entrypoint.sh",
		}
	}
	return nil
}

func (statsd *Statsd) buildPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{Name: capability.StatsdIngestTargetPort, ContainerPort: capability.StatsdIngestPort},
		{Name: statsdProbesPortName, ContainerPort: statsdProbesPort},
	}
}

func (statsd *Statsd) buildVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: dataSourceStartupArguments, MountPath: dataSourceStartupArgsMountPoint},
		{Name: dataSourceAuthToken, MountPath: dataSourceAuthTokenMountPoint},
		{Name: dataSourceMetadata, MountPath: dataSourceMetadataMountPoint},
		{Name: dataSourceStatsdLogs, MountPath: statsdLogsDir},
	}
}

func (statsd *Statsd) buildEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: envStatsdStartupArgsPath, Value: dataSourceStartupArgsMountPoint + "/statsd.process.json"},
		{Name: envDataSourceProbeServerPort, Value: fmt.Sprintf("%d", statsdProbesPort)},
		{Name: envStatsdMetadata, Value: dataSourceMetadataMountPoint},
		{Name: envDataSourceLogFile, Value: statsdLogsDir + "/dynatracesourcestatsd.log"},
	}
}

func (statsd *Statsd) buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               address_of.Scalar(false),
		AllowPrivilegeEscalation: address_of.Scalar(false),
		ReadOnlyRootFilesystem:   address_of.Scalar(true),

		RunAsNonRoot: address_of.Scalar(true),
		RunAsUser:    address_of.Scalar(kubeobjects.UnprivilegedUser),
		RunAsGroup:   address_of.Scalar(kubeobjects.UnprivilegedGroup),

		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"all",
			},
		},
	}
}

var _ dynatracev1beta1.ResourceRequirementer = (*Statsd)(nil)

func (statsd *Statsd) Limits(resourceName corev1.ResourceName) *resource.Quantity {
	return statsd.stsProperties.FeatureStatsdResourcesLimits(resourceName)
}

func (statsd *Statsd) Requests(resourceName corev1.ResourceName) *resource.Quantity {
	return statsd.stsProperties.FeatureStatsdResourcesRequests(resourceName)
}

func (statsd *Statsd) buildResourceRequirements() corev1.ResourceRequirements {
	return dynatracev1beta1.BuildResourceRequirements(statsd)
}
