package statefulset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const eecIngestPortName = "eec-http"
const eecIngestPort = 14599

const activeGateInternalCommunicationPort = 9999

const (
	eecAuthToken = "auth-tokens"

	dataSourceStartupArguments = "eec-ds-shared"
	dataSourceAuthToken        = "dsauthtokendir"
)

var _ kubeobjects.ContainerBuilder = (*ExtensionController)(nil)

type ExtensionController struct {
	stsProperties *statefulSetProperties
}

func NewExtensionController(stsProperties *statefulSetProperties) *ExtensionController {
	return &ExtensionController{
		stsProperties: stsProperties,
	}
}

func (eec *ExtensionController) BuildContainer() corev1.Container {
	return corev1.Container{
		Name:            consts.EecContainerName,
		Image:           eec.stsProperties.DynaKube.ActiveGateImage(),
		ImagePullPolicy: corev1.PullAlways,
		Env:             eec.buildEnvs(),
		VolumeMounts:    eec.buildVolumeMounts(),
		Command:         eec.buildCommand(),
		Ports:           eec.buildPorts(),
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.IntOrString{IntVal: eecIngestPort},
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       15,
			FailureThreshold:    3,
			SuccessThreshold:    1,
			TimeoutSeconds:      1,
		},
	}
}

func (eec *ExtensionController) BuildVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: eecAuthToken,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: dataSourceStartupArguments,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: dataSourceAuthToken,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func (eec *ExtensionController) buildPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{Name: eecIngestPortName, ContainerPort: eecIngestPort},
	}
}

func (eec *ExtensionController) buildCommand() []string {
	if eec.stsProperties.DynaKube.FeatureUseActiveGateImageForStatsD() {
		return []string{
			"/bin/bash", "/dt/eec/entrypoint.sh",
		}
	}
	return nil
}

func (eec *ExtensionController) buildVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: eecAuthToken, MountPath: "/var/lib/dynatrace/gateway/config"},
		{Name: dataSourceStartupArguments, MountPath: "/mnt/dsexecargs"},
		{Name: dataSourceAuthToken, MountPath: "/var/lib/dynatrace/remotepluginmodule/agent/runtime/datasources"},
		{Name: dataSourceMetadata, MountPath: "/opt/dynatrace/remotepluginmodule/agent/datasources/statsd", ReadOnly: true},
	}
}

func (eec *ExtensionController) buildEnvs() []corev1.EnvVar {
	tenantId, err := eec.stsProperties.TenantUUID()
	if err != nil {
		log.Error(err, "Problem getting tenant id from api url")
	}
	return []corev1.EnvVar{
		{Name: "TenantId", Value: tenantId},
		{Name: "ServerUrl", Value: fmt.Sprintf("https://localhost:%d/communication", activeGateInternalCommunicationPort)},
		{Name: "EecIngestPort", Value: fmt.Sprintf("%d", eecIngestPort)},
	}
}
