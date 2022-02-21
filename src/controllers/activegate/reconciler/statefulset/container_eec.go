package statefulset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address_of"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const eecIngestPortName = "eec-http"
const eecIngestPort = 14599
const extensionsLogsDir = "/var/lib/dynatrace/remotepluginmodule/log/extensions"

const activeGateInternalCommunicationPort = 9999

const (
	eecAuthToken = "auth-tokens"

	dataSourceStartupArguments = "eec-ds-shared"
	dataSourceAuthToken        = "dsauthtokendir"
	eecLogs                    = "extensions-logs"
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
		Image:           eec.image(),
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
		SecurityContext: eec.buildSecurityContext(),
		Resources:       eec.buildResourceRequirements(),
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
		{
			Name: eecLogs,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func (eec *ExtensionController) image() string {
	if eec.stsProperties.FeatureUseActiveGateImageForStatsd() {
		return eec.stsProperties.ActiveGateImage()
	}
	return eec.stsProperties.EecImage()
}

func (eec *ExtensionController) buildPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{Name: eecIngestPortName, ContainerPort: eecIngestPort},
	}
}

func (eec *ExtensionController) buildCommand() []string {
	if eec.stsProperties.DynaKube.FeatureUseActiveGateImageForStatsd() {
		return []string{
			"/bin/bash", "/dt/eec/entrypoint.sh",
		}
	}
	return nil
}

func (eec *ExtensionController) buildVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: eecAuthToken, MountPath: activeGateConfigDir},
		{Name: dataSourceStartupArguments, MountPath: dataSourceStartupArgsMountPoint},
		{Name: dataSourceAuthToken, MountPath: dataSourceAuthTokenMountPoint},
		{Name: dataSourceMetadata, MountPath: statsdMetadataMountPoint, ReadOnly: true},
		{Name: eecLogs, MountPath: extensionsLogsDir},
		{Name: dataSourceStatsdLogs, MountPath: statsDLogsDir, ReadOnly: true},
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

func (eec *ExtensionController) buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               address_of.Bool(false),
		AllowPrivilegeEscalation: address_of.Bool(false),
		ReadOnlyRootFilesystem:   address_of.Bool(false),

		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"all",
			},
		},
	}
}

var _ v1beta1.ResourceRequirementer = (*ExtensionController)(nil)

func (eec *ExtensionController) Limits(resourceName corev1.ResourceName) *resource.Quantity {
	return eec.stsProperties.FeatureEecResourcesLimits(resourceName)
}

func (eec *ExtensionController) Requests(resourceName corev1.ResourceName) *resource.Quantity {
	return eec.stsProperties.FeatureEecResourcesRequests(resourceName)
}

func (eec *ExtensionController) buildResourceRequirements() corev1.ResourceRequirements {
	return v1beta1.BuildResourceRequirements(eec)
}
