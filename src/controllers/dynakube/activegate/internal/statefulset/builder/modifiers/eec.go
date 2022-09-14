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

var _ builder.Modifier = ExtensionControllerModifier{}

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

type ExtensionControllerModifier struct {
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
}

func NewExtensionControllerModifier(dynakube dynatracev1beta1.DynaKube, capability capability.Capability) ExtensionControllerModifier {
	return ExtensionControllerModifier{
		dynakube:   dynakube,
		capability: capability,
	}
}

func (eec ExtensionControllerModifier) Enabled() bool {
	return eec.dynakube.IsStatsdActiveGateEnabled()
}

func (eec ExtensionControllerModifier) Modify(sts *appsv1.StatefulSet) {
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, eec.buildContainer())
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, eec.getVolumes(sts.Spec.Template.Spec.Volumes)...)

	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, eec.getActiveGateVolumeMounts(baseContainer.VolumeMounts)...)

}

func (eec ExtensionControllerModifier) getActiveGateVolumeMounts(presentMounts []corev1.VolumeMount) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{{Name: eecLogs, MountPath: extensionsLogsDir + "/eec", ReadOnly: true}}
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

func (eec ExtensionControllerModifier) getVolumes(presentVolumes []corev1.Volume) []corev1.Volume {
	volumes := eec.buildVolumes()
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

func (eec ExtensionControllerModifier) buildContainer() corev1.Container {
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

func (eec ExtensionControllerModifier) buildVolumes() []corev1.Volume {
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

	if len(eec.dynakube.Name) > 0 && len(eec.capability.ShortName()) > 0 {
		eecConfigMap := corev1.Volume{
			Name: eecConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: capability.BuildEecConfigMapName(eec.dynakube.Name, eec.capability.ShortName()),
					},
				},
			},
		}
		volumes = append(volumes, eecConfigMap)
	} else {
		err := fmt.Errorf("empty instance or module name not allowed (instance: %s, module: %s)", eec.dynakube.Name, eec.capability.ShortName())
		log.Info("problem building EEC config map name", err, err.Error())
	}

	return volumes
}

func (eec ExtensionControllerModifier) image() string {
	if eec.dynakube.FeatureUseActiveGateImageForStatsd() {
		return eec.dynakube.ActiveGateImage()
	}
	return eec.dynakube.EecImage()
}

func (eec ExtensionControllerModifier) buildPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{Name: eecIngestPortName, ContainerPort: eecIngestPort},
	}
}

func (eec ExtensionControllerModifier) buildCommand() []string {
	if eec.dynakube.FeatureUseActiveGateImageForStatsd() {
		return []string{
			"/bin/bash", "/dt/eec/entrypoint.sh",
		}
	}
	return nil
}

func (eec ExtensionControllerModifier) buildVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: consts.GatewayConfigVolumeName, ReadOnly: true, MountPath: consts.GatewayConfigMountPoint},
		{Name: dataSourceStartupArguments, MountPath: dataSourceStartupArgsMountPoint},
		{Name: dataSourceAuthToken, MountPath: dataSourceAuthTokenMountPoint},
		{Name: dataSourceMetadata, MountPath: statsdMetadataMountPoint, ReadOnly: true},
		{Name: eecLogs, MountPath: extensionsLogsDir},
		{Name: dataSourceStatsdLogs, MountPath: statsdLogsDir, ReadOnly: true},
		{Name: eecConfig, MountPath: extensionsRuntimeDir},
	}
}

func (eec ExtensionControllerModifier) buildEnvs() []corev1.EnvVar {
	tenantId, err := eec.dynakube.TenantUUID()
	if err != nil {
		log.Error(err, "Problem getting tenant id from api url")
	}
	return []corev1.EnvVar{
		{Name: envTenantId, Value: tenantId},
		{Name: envServerUrl, Value: fmt.Sprintf("https://localhost:%d/communication", activeGateInternalCommunicationPort)},
		{Name: envEecIngestPort, Value: fmt.Sprintf("%d", eecIngestPort)},
	}
}

func (eec ExtensionControllerModifier) buildSecurityContext() *corev1.SecurityContext {
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

var _ dynatracev1beta1.ResourceRequirementer = (*ExtensionControllerModifier)(nil)

func (eec ExtensionControllerModifier) Limits(resourceName corev1.ResourceName) *resource.Quantity {
	return eec.dynakube.FeatureEecResourcesLimits(resourceName)
}

func (eec ExtensionControllerModifier) Requests(resourceName corev1.ResourceName) *resource.Quantity {
	return eec.dynakube.FeatureEecResourcesRequests(resourceName)
}

func (eec ExtensionControllerModifier) buildResourceRequirements() corev1.ResourceRequirements {
	return dynatracev1beta1.BuildResourceRequirements(eec)
}
