package statefulset

import (
	"fmt"
	"regexp"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const eecIngestPortName = "eec-http"
const eecIngestPort = 14599
const extensionsLogsDir = "/var/lib/dynatrace/remotepluginmodule/log/extensions"
const extensionsRuntimeDir = "/var/lib/dynatrace/remotepluginmodule/agent/conf/runtime"

const activeGateInternalCommunicationPort = 9999

const (
	dataSourceStartupArguments = "eec-ds-shared"
	dataSourceAuthToken        = "dsauthtokendir"
	eecLogs                    = "extensions-logs"
	eecConfig                  = "eec-config"

	envTenantId      = "TenantId"
	envServerUrl     = "ServerUrl"
	envEecIngestPort = "EecIngestPort"
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
		Name:            EecContainerName,
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
	volumes := []corev1.Volume{
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

	if len(eec.stsProperties.Name) > 0 && len(eec.stsProperties.feature) > 0 {
		eecConfigMap := corev1.Volume{
			Name: eecConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: BuildEecConfigMapName(eec.stsProperties.Name, eec.stsProperties.feature),
					},
				},
			},
		}
		volumes = append(volumes, eecConfigMap)
	} else {
		err := fmt.Errorf("empty instance or module name not allowed (instance: %s, module: %s)", eec.stsProperties.Name, eec.stsProperties.feature)
		log.Info("problem building EEC config map name", err, err.Error())
	}

	return volumes
}

func BuildEecConfigMapName(instanceName string, module string) string {
	return regexp.MustCompile(`[^\w\-]`).ReplaceAllString(instanceName+"-"+module+"-eec-config", "_")
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
		{Name: GatewayConfigVolumeName, ReadOnly: true, MountPath: GatewayConfigMountPoint},
		{Name: dataSourceStartupArguments, MountPath: dataSourceStartupArgsMountPoint},
		{Name: dataSourceAuthToken, MountPath: dataSourceAuthTokenMountPoint},
		{Name: dataSourceMetadata, MountPath: statsdMetadataMountPoint, ReadOnly: true},
		{Name: eecLogs, MountPath: extensionsLogsDir},
		{Name: dataSourceStatsdLogs, MountPath: statsdLogsDir, ReadOnly: true},
		{Name: eecConfig, MountPath: extensionsRuntimeDir},
	}
}

func (eec *ExtensionController) buildEnvs() []corev1.EnvVar {
	tenantId, err := eec.stsProperties.TenantUUID()
	if err != nil {
		log.Error(err, "Problem getting tenant id from api url")
	}
	return []corev1.EnvVar{
		{Name: envTenantId, Value: tenantId},
		{Name: envServerUrl, Value: fmt.Sprintf("https://localhost:%d/communication", activeGateInternalCommunicationPort)},
		{Name: envEecIngestPort, Value: fmt.Sprintf("%d", eecIngestPort)},
	}
}

func (eec *ExtensionController) buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               address.Of(false),
		AllowPrivilegeEscalation: address.Of(false),
		ReadOnlyRootFilesystem:   address.Of(false),

		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}
}

var _ dynatracev1beta1.ResourceRequirementer = (*ExtensionController)(nil)

func (eec *ExtensionController) Limits(resourceName corev1.ResourceName) *resource.Quantity {
	return eec.stsProperties.FeatureEecResourcesLimits(resourceName)
}

func (eec *ExtensionController) Requests(resourceName corev1.ResourceName) *resource.Quantity {
	return eec.stsProperties.FeatureEecResourcesRequests(resourceName)
}

func (eec *ExtensionController) buildResourceRequirements() corev1.ResourceRequirements {
	return dynatracev1beta1.BuildResourceRequirements(eec)
}
