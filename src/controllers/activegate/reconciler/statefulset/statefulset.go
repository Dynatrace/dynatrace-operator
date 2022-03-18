package statefulset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/agproxysecret"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/events"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address_of"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	serviceAccountPrefix   = "dynatrace-"
	tenantSecretVolumeName = "ag-tenant-secret"

	annotationCustomPropsHash             = dynatracev1beta1.InternalFlagPrefix + "custom-properties-hash"
	annotationActiveGateContainerAppArmor = "container.apparmor.security.beta.kubernetes.io/" + capability.ActiveGateContainerName

	dtServer             = "DT_SERVER"
	dtTenant             = "DT_TENANT"
	dtCapabilities       = "DT_CAPABILITIES"
	dtIdSeedNamespace    = "DT_ID_SEED_NAMESPACE"
	dtIdSeedClusterId    = "DT_ID_SEED_K8S_CLUSTER_ID"
	dtNetworkZone        = "DT_NETWORK_ZONE"
	dtGroup              = "DT_GROUP"
	dtDeploymentMetadata = "DT_DEPLOYMENT_METADATA"

	dataSourceStartupArgsMountPoint = "/mnt/dsexecargs"
	dataSourceAuthTokenMountPoint   = "/var/lib/dynatrace/remotepluginmodule/agent/runtime/datasources"
	dataSourceMetadataMountPoint    = "/mnt/dsmetadata"
	statsdMetadataMountPoint        = "/opt/dynatrace/remotepluginmodule/agent/datasources/statsd"
	tenantTokenMountPoint           = "/var/lib/dynatrace/secrets/tokens/tenant-token"

	DeploymentTypeActiveGate = "active_gate"
)

type statefulSetProperties struct {
	*dynatracev1beta1.DynaKube
	*dynatracev1beta1.CapabilityProperties
	customPropertiesHash    string
	kubeSystemUID           types.UID
	feature                 string
	capabilityName          string
	serviceAccountOwner     string
	OnAfterCreateListener   []events.StatefulSetEvent
	initContainersTemplates []corev1.Container
	containerVolumeMounts   []corev1.VolumeMount
	volumes                 []corev1.Volume
}

func NewStatefulSetProperties(instance *dynatracev1beta1.DynaKube, capabilityProperties *dynatracev1beta1.CapabilityProperties, kubeSystemUID types.UID,
	customPropertiesHash string, feature string, capabilityName string, serviceAccountOwner string,
	initContainers []corev1.Container, containerVolumeMounts []corev1.VolumeMount, volumes []corev1.Volume) *statefulSetProperties {

	if serviceAccountOwner == "" {
		serviceAccountOwner = feature
	}

	return &statefulSetProperties{
		DynaKube:                instance,
		CapabilityProperties:    capabilityProperties,
		customPropertiesHash:    customPropertiesHash,
		kubeSystemUID:           kubeSystemUID,
		feature:                 feature,
		capabilityName:          capabilityName,
		serviceAccountOwner:     serviceAccountOwner,
		OnAfterCreateListener:   []events.StatefulSetEvent{},
		initContainersTemplates: initContainers,
		containerVolumeMounts:   containerVolumeMounts,
		volumes:                 volumes,
	}
}

func CreateStatefulSet(stsProperties *statefulSetProperties) (*appsv1.StatefulSet, error) {
	appLabels := kubeobjects.NewAppLabels(kubeobjects.ActiveGateComponentLabel, stsProperties.DynaKube.Name,
		stsProperties.feature, stsProperties.Status.ActiveGate.Version)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        stsProperties.Name + "-" + stsProperties.feature,
			Namespace:   stsProperties.Namespace,
			Labels:      appLabels.BuildLabels(),
			Annotations: map[string]string{},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            stsProperties.Replicas,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector:            &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: appLabels.BuildLabels(),
					Annotations: map[string]string{
						annotationCustomPropsHash: stsProperties.customPropertiesHash,
					},
				},
				Spec: buildTemplateSpec(stsProperties),
			},
		}}

	if stsProperties.DynaKube.FeatureActiveGateAppArmor() {
		sts.Spec.Template.ObjectMeta.Annotations[annotationActiveGateContainerAppArmor] = "runtime/default"
	}

	for _, onAfterCreateListener := range stsProperties.OnAfterCreateListener {
		onAfterCreateListener(sts)
	}

	hash, err := kubeobjects.GenerateHash(sts)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sts.ObjectMeta.Annotations[kubeobjects.AnnotationHash] = hash
	return sts, nil
}

func getContainerBuilders(stsProperties *statefulSetProperties) []kubeobjects.ContainerBuilder {
	if stsProperties.NeedsStatsd() {
		return []kubeobjects.ContainerBuilder{
			NewExtensionController(stsProperties),
			NewStatsd(stsProperties),
		}
	}
	return nil
}

func buildTemplateSpec(stsProperties *statefulSetProperties) corev1.PodSpec {
	extraContainerBuilders := getContainerBuilders(stsProperties)
	podSpec := corev1.PodSpec{
		Containers:         buildContainers(stsProperties, extraContainerBuilders),
		InitContainers:     buildInitContainers(stsProperties),
		NodeSelector:       stsProperties.CapabilityProperties.NodeSelector,
		ServiceAccountName: determineServiceAccountName(stsProperties),
		Affinity:           affinity(),
		Tolerations:        stsProperties.Tolerations,
		Volumes:            buildVolumes(stsProperties, extraContainerBuilders),
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: stsProperties.PullSecret()},
		},
		PriorityClassName:         stsProperties.DynaKube.Spec.ActiveGate.PriorityClassName,
		TopologySpreadConstraints: stsProperties.TopologySpreadConstraints,
	}
	if dnsPolicy := buildDNSPolicy(stsProperties); dnsPolicy != "" {
		podSpec.DNSPolicy = dnsPolicy
	}
	return podSpec
}

func buildDNSPolicy(stsProperties *statefulSetProperties) corev1.DNSPolicy {
	if stsProperties.ActiveGateMode() {
		return stsProperties.Spec.ActiveGate.DNSPolicy
	}
	return ""
}

func buildInitContainers(stsProperties *statefulSetProperties) []corev1.Container {
	ics := stsProperties.initContainersTemplates

	for idx := range ics {
		ics[idx].Image = stsProperties.DynaKube.ActiveGateImage()
		ics[idx].Resources = stsProperties.CapabilityProperties.Resources
	}

	return ics
}

func buildContainers(stsProperties *statefulSetProperties, extraContainerBuilders []kubeobjects.ContainerBuilder) []corev1.Container {
	containers := []corev1.Container{
		buildActiveGateContainer(stsProperties),
	}

	for _, containerBuilder := range extraContainerBuilders {
		containers = append(containers,
			containerBuilder.BuildContainer(),
		)
	}
	return containers
}

func buildActiveGateContainer(stsProperties *statefulSetProperties) corev1.Container {
	readOnlyFs := stsProperties.FeatureActiveGateReadOnlyFilesystem()

	return corev1.Container{
		Name:            capability.ActiveGateContainerName,
		Image:           stsProperties.DynaKube.ActiveGateImage(),
		Resources:       stsProperties.CapabilityProperties.Resources,
		ImagePullPolicy: corev1.PullAlways,
		Env:             buildEnvs(stsProperties),
		VolumeMounts:    buildVolumeMounts(stsProperties),
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/rest/health",
					Port:   intstr.IntOrString{IntVal: 9999},
					Scheme: "HTTPS",
				},
			},
			InitialDelaySeconds: 90,
			PeriodSeconds:       15,
			FailureThreshold:    3,
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged:               address_of.Scalar(false),
			AllowPrivilegeEscalation: address_of.Scalar(false),
			ReadOnlyRootFilesystem:   &readOnlyFs,
			RunAsNonRoot:             address_of.Scalar(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"all",
				},
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
	}
}

func buildVolumes(stsProperties *statefulSetProperties, extraContainerBuilders []kubeobjects.ContainerBuilder) []corev1.Volume {
	var volumes []corev1.Volume

	if !stsProperties.DynaKube.FeatureDisableActivegateRawImage() {
		volumes = append(volumes, corev1.Volume{
			Name: tenantSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: stsProperties.AGTenantSecret(),
				},
			},
		},
		)
	}

	if !isCustomPropertiesNilOrEmpty(stsProperties.CustomProperties) {
		valueFrom := determineCustomPropertiesSource(stsProperties)
		volumes = append(volumes,
			corev1.Volume{
				Name: customproperties.VolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: valueFrom,
						Items: []corev1.KeyToPath{
							{Key: customproperties.DataKey, Path: customproperties.DataPath},
						}}},
			},
		)
	}

	for _, containerBuilder := range extraContainerBuilders {
		volumes = append(volumes,
			containerBuilder.BuildVolumes()...,
		)
	}

	volumes = append(volumes, stsProperties.volumes...)

	if stsProperties.NeedsActiveGateProxy() {
		volumes = append(volumes, buildProxyVolumes()...)
	}

	volumes = append(volumes, buildActiveGateVolumes(stsProperties)...)

	return volumes
}

func buildActiveGateVolumes(stsProperties *statefulSetProperties) []corev1.Volume {
	var volumes []corev1.Volume
	if stsProperties.FeatureActiveGateReadOnlyFilesystem() || stsProperties.NeedsStatsd() {
		volumes = append(volumes, corev1.Volume{
			Name: capability.ActiveGateGatewayConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	if stsProperties.FeatureActiveGateReadOnlyFilesystem() {
		volumes = append(volumes,
			corev1.Volume{
				Name: capability.ActiveGateGatewayTempVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			corev1.Volume{
				Name: capability.ActiveGateGatewayDataVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			corev1.Volume{
				Name: capability.ActiveGateLogVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			corev1.Volume{
				Name: capability.ActiveGateTmpVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)

		if stsProperties.HasActiveGateCaCert() {
			volumes = append(volumes,
				corev1.Volume{
					Name: capability.ActiveGateGatewaySslVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			)
		}
	}

	return volumes
}

func buildProxyVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: InternalProxySecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: agproxysecret.BuildProxySecretName(),
				},
			},
		},
	}
}

func determineCustomPropertiesSource(stsProperties *statefulSetProperties) string {
	if stsProperties.CustomProperties.ValueFrom == "" {
		return fmt.Sprintf("%s-%s-%s", stsProperties.Name, stsProperties.serviceAccountOwner, customproperties.Suffix)
	}
	return stsProperties.CustomProperties.ValueFrom
}

func buildVolumeMounts(stsProperties *statefulSetProperties) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	if !isCustomPropertiesNilOrEmpty(stsProperties.CustomProperties) {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      customproperties.VolumeName,
			MountPath: customproperties.MountPath,
			SubPath:   customproperties.DataPath,
		})
	}

	if stsProperties.NeedsStatsd() {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{Name: eecLogs, MountPath: extensionsLogsDir + "/eec", ReadOnly: true},
			corev1.VolumeMount{Name: dataSourceStatsdLogs, MountPath: extensionsLogsDir + "/statsd", ReadOnly: true},
		)
	}

	volumeMounts = append(volumeMounts, stsProperties.containerVolumeMounts...)

	if stsProperties.NeedsActiveGateProxy() {
		volumeMounts = append(volumeMounts, buildProxyMounts()...)
	}

	if !stsProperties.DynaKube.FeatureDisableActivegateRawImage() {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      tenantSecretVolumeName,
			ReadOnly:  true,
			MountPath: tenantTokenMountPoint,
			SubPath:   activegate.TenantTokenName,
		},
		)
	}

	volumeMounts = append(volumeMounts, buildActiveGateVolumeMounts(stsProperties)...)

	return volumeMounts
}

func buildActiveGateVolumeMounts(stsProperties *statefulSetProperties) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount
	if stsProperties.FeatureActiveGateReadOnlyFilesystem() || stsProperties.NeedsStatsd() {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			ReadOnly:  false,
			Name:      capability.ActiveGateGatewayConfigVolumeName,
			MountPath: capability.ActiveGateGatewayConfigMountPoint,
		})
	}
	if stsProperties.FeatureActiveGateReadOnlyFilesystem() {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				ReadOnly:  false,
				Name:      capability.ActiveGateGatewayTempVolumeName,
				MountPath: capability.ActiveGateGatewayTempMountPoint,
			},
			corev1.VolumeMount{
				ReadOnly:  false,
				Name:      capability.ActiveGateGatewayDataVolumeName,
				MountPath: capability.ActiveGateGatewayDataMountPoint,
			},
			corev1.VolumeMount{
				ReadOnly:  false,
				Name:      capability.ActiveGateLogVolumeName,
				MountPath: capability.ActiveGateLogMountPoint,
			},
			corev1.VolumeMount{
				ReadOnly:  false,
				Name:      capability.ActiveGateTmpVolumeName,
				MountPath: capability.ActiveGateTmpMountPoint,
			})

		if stsProperties.HasActiveGateCaCert() {
			volumeMounts = append(volumeMounts,
				corev1.VolumeMount{
					ReadOnly:  false,
					Name:      capability.ActiveGateGatewaySslVolumeName,
					MountPath: capability.ActiveGateGatewaySslMountPoint,
				},
			)
		}
	}
	return volumeMounts
}

func buildProxyMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretHostMountPath,
			SubPath:   InternalProxySecretHost,
		},
		{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretPortMountPath,
			SubPath:   InternalProxySecretPort,
		},
		{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretUsernameMountPath,
			SubPath:   InternalProxySecretUsername,
		},
		{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretPasswordMountPath,
			SubPath:   InternalProxySecretPassword,
		},
	}
}

func buildEnvs(stsProperties *statefulSetProperties) []corev1.EnvVar {
	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(string(stsProperties.kubeSystemUID), DeploymentTypeActiveGate)

	envs := []corev1.EnvVar{
		{Name: dtCapabilities, Value: stsProperties.capabilityName},
		{Name: dtIdSeedNamespace, Value: stsProperties.Namespace},
		{Name: dtIdSeedClusterId, Value: string(stsProperties.kubeSystemUID)},
		{Name: dtDeploymentMetadata, Value: deploymentMetadata.AsString()},
	}

	if !stsProperties.DynaKube.FeatureDisableActivegateRawImage() {
		envs = append(envs,
			communicationEndpointEnvVar(stsProperties),
			tenantUuidNameEnvVar(stsProperties))
	}

	envs = append(envs, stsProperties.Env...)

	if stsProperties.Group != "" {
		envs = append(envs, corev1.EnvVar{Name: dtGroup, Value: stsProperties.Group})
	}
	if stsProperties.Spec.NetworkZone != "" {
		envs = append(envs, corev1.EnvVar{Name: dtNetworkZone, Value: stsProperties.Spec.NetworkZone})
	}

	return envs
}

func tenantUuidNameEnvVar(stsProperties *statefulSetProperties) corev1.EnvVar {
	return corev1.EnvVar{
		Name: dtTenant,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: stsProperties.AGTenantSecret(),
				},
				Key: activegate.TenantUuidName,
			},
		},
	}
}

func communicationEndpointEnvVar(stsProperties *statefulSetProperties) corev1.EnvVar {
	return corev1.EnvVar{
		Name: dtServer,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: stsProperties.AGTenantSecret(),
				},
				Key: activegate.CommunicationEndpointsName,
			},
		},
	}
}

func determineServiceAccountName(stsProperties *statefulSetProperties) string {
	return serviceAccountPrefix + stsProperties.serviceAccountOwner
}

func isCustomPropertiesNilOrEmpty(customProperties *dynatracev1beta1.DynaKubeValueSource) bool {
	return customProperties == nil ||
		(customProperties.Value == "" &&
			customProperties.ValueFrom == "")
}
