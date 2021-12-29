package statefulset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const statsdProbesPortName = "statsd-probes"
const statsdProbesPort = 14999
const statsDLogsDir = extensionsLogsDir + "/datasources-statsd"

const (
	dataSourceMetadata = "ds-metadata"
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
			Name: "statsd-logs",
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
		{Name: consts.StatsdIngestTargetPort, ContainerPort: consts.StatsdIngestPort},
		{Name: statsdProbesPortName, ContainerPort: statsdProbesPort},
	}
}

func (statsd *Statsd) buildVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: dataSourceStartupArguments, MountPath: "/mnt/dsexecargs"},
		{Name: dataSourceAuthToken, MountPath: "/var/lib/dynatrace/remotepluginmodule/agent/runtime/datasources"},
		{Name: dataSourceMetadata, MountPath: "/mnt/dsmetadata"},
		{Name: "statsd-logs", MountPath: statsDLogsDir},
	}
}

func (statsd *Statsd) buildEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "StatsdExecArgsPath", Value: "/mnt/dsexecargs/statsd.process.json"},
		{Name: "ProbeServerPort", Value: fmt.Sprintf("%d", statsdProbesPort)},
		{Name: "StatsdMetadataDir", Value: "/mnt/dsmetadata"},
		{Name: "DsLogFile", Value: statsDLogsDir + "/dynatracesourcestatsd.log"},
	}
}
