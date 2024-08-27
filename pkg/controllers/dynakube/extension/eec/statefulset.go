package eec

import (
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/hash"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/utils"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	runtimePersistentVolumeClaimName = dynakube.ExtensionsExecutionControllerStatefulsetName + "-runtime"
	containerName                    = "extensions-controller"
	collectorPort                    = int32(14599)
	serviceAccountName               = "dynatrace-extensions-controller"

	// Env variable names
	envTenantId                     = "TenantId"
	envServerUrl                    = "ServerUrl"
	envEecTokenPath                 = "EecTokenPath"
	envEecIngestPort                = "EecIngestPort"
	envExtensionsModuleExecPathName = "ExtensionsModuleExecPath"
	envDsInstallDirName             = "DsInstallDir"
	envK8sClusterId                 = "K8sClusterUID"
	envActiveGateTrustedCertName    = "ActiveGateTrustedCert"
	envK8sExtServiceUrl             = "K8sExtServiceUrl"
	envHttpsCertPathPem             = "HttpsCertPathPem"
	envHttpsPrivKeyPathPem          = "HttpsPrivKeyPathPem"
	envDSTokenPath                  = "DSTokenPath"
	envRuntimeConfigMountPath       = "RuntimeConfigMountPath"
	// Env variable values
	envExtensionsModuleExecPath = "/opt/dynatrace/remotepluginmodule/agent/lib64/extensionsmodule"
	envDsInstallDir             = "/opt/dynatrace/remotepluginmodule/agent/datasources"
	envActiveGateTrustedCert    = activeGateTrustedCertMountPath + "/" + activeGateTrustedCertSecretKeyPath
	envEecHttpsCertPathPem      = httpsCertMountPath + "/" + consts.TLSCrtDataName
	envEecHttpsPrivKeyPathPem   = httpsCertMountPath + "/" + consts.TLSKeyDataName
	// Volume names and paths
	eecTokenMountPath                  = "/var/lib/dynatrace/remotepluginmodule/secrets/tokens"
	logMountPath                       = "/var/lib/dynatrace/remotepluginmodule/log"
	runtimeVolumeName                  = "agent-runtime"
	runtimeMountPath                   = "/var/lib/dynatrace/remotepluginmodule/agent/runtime"
	configurationVolumeName            = "runtime-configuration"
	configurationMountPath             = "/var/lib/dynatrace/remotepluginmodule/agent/conf"
	customConfigVolumeName             = "custom-config"
	customConfigMountPath              = "/var/lib/dynatrace/remotepluginmodule/secrets/config"
	activeGateTrustedCertVolumeName    = "server-certs"
	activeGateTrustedCertMountPath     = "/var/lib/dynatrace/remotepluginmodule/secrets/ag"
	activeGateTrustedCertSecretKeyPath = "server.crt"
	httpsCertVolumeName                = "https-certs"
	httpsCertMountPath                 = "/var/lib/dynatrace/remotepluginmodule/secrets/https"
	extensionsControllerTlsSecretName  = "extensions-controller-tls"
	dsTokenPath                        = "/var/lib/dynatrace/remotepluginmodule/secrets/dsauthtoken"
	runtimeConfigurationFilename       = "runtimeConfiguration"

	// misc
	logVolumeName = "log"
)

func (r *reconciler) createOrUpdateStatefulset(ctx context.Context) error {
	appLabels := buildAppLabels(r.dk.Name)
	desiredSts, err := statefulset.Build(r.dk, dynakube.ExtensionsExecutionControllerStatefulsetName, buildContainer(r.dk),
		statefulset.SetReplicas(1),
		statefulset.SetPodManagementPolicy(appsv1.ParallelPodManagement),
		statefulset.SetAllLabels(appLabels.BuildLabels(), appLabels.BuildMatchLabels(), appLabels.BuildLabels(), r.dk.Spec.Templates.ExtensionExecutionController.Labels),
		statefulset.SetAllAnnotations(nil, r.dk.Spec.Templates.ExtensionExecutionController.Annotations),
		statefulset.SetAffinity(buildAffinity()),
		statefulset.SetTolerations(r.dk.Spec.Templates.ExtensionExecutionController.Tolerations),
		statefulset.SetTopologySpreadConstraints(utils.BuildTopologySpreadConstraints(r.dk.Spec.Templates.ExtensionExecutionController.TopologySpreadConstraints, appLabels)),
		statefulset.SetServiceAccount(serviceAccountName),
		statefulset.SetSecurityContext(buildPodSecurityContext()),
		statefulset.SetUpdateStrategy(utils.BuildUpdateStrategy()),
		setImagePullSecrets(r.dk.ImagePullSecretReferences()),
		setVolumes(r.dk),
	)

	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, err)

		return err
	}

	if err := hash.SetHash(desiredSts); err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, err)

		return err
	}

	_, err = statefulset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, desiredSts)
	if err != nil {
		log.Info("failed to create/update " + dynakube.ExtensionsExecutionControllerStatefulsetName + " statefulset")
		conditions.SetKubeApiError(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, err)

		return err
	}

	conditions.SetStatefulSetCreated(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, desiredSts.Name)

	return nil
}

func buildAppLabels(dynakubeName string) *labels.AppLabels {
	// TODO: when version is available
	version := "0.0.0"

	return labels.NewAppLabels(labels.ExtensionComponentLabel, dynakubeName, labels.ExtensionComponentLabel, version)
}

func buildAffinity() corev1.Affinity {
	return corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: node.AffinityNodeRequirementForSupportedArches(),
					},
				},
			},
		},
	}
}

func setImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}
}

func buildContainer(dk *dynakube.DynaKube) corev1.Container {
	return corev1.Container{
		Name:            containerName,
		Image:           dk.Spec.Templates.ExtensionExecutionController.ImageRef.Repository + ":" + dk.Spec.Templates.ExtensionExecutionController.ImageRef.Tag,
		ImagePullPolicy: corev1.PullAlways,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/readyz",
					Port:   intstr.IntOrString{IntVal: collectorPort},
					Scheme: "HTTPS",
				},
			},
			InitialDelaySeconds: 15,
			PeriodSeconds:       15,
			FailureThreshold:    3,
			TimeoutSeconds:      2,
			SuccessThreshold:    1,
		},
		SecurityContext: buildSecurityContext(),
		Ports: []corev1.ContainerPort{
			{
				Name:          consts.ExtensionsCollectorTargetPortName,
				ContainerPort: collectorPort,
			},
		},
		Env:          buildContainerEnvs(dk),
		Resources:    dk.Spec.Templates.ExtensionExecutionController.Resources,
		VolumeMounts: buildContainerVolumeMounts(dk),
	}
}

func buildSecurityContext() *corev1.SecurityContext {
	userGroupId := int64(1001)

	return &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		Privileged:               address.Of(false),
		RunAsUser:                &userGroupId,
		RunAsGroup:               &userGroupId,
		RunAsNonRoot:             address.Of(true),
		ReadOnlyRootFilesystem:   address.Of(true),
		AllowPrivilegeEscalation: address.Of(false),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func buildPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func buildContainerEnvs(dk *dynakube.DynaKube) []corev1.EnvVar {
	containerEnvs := []corev1.EnvVar{
		{Name: envTenantId, Value: dk.Status.ActiveGate.ConnectionInfoStatus.TenantUUID},
		{Name: envServerUrl, Value: buildActiveGateServiceName(dk) + "." + dk.Namespace + ".svc.cluster.local:443"},
		{Name: envEecTokenPath, Value: eecTokenMountPath + "/" + consts.EecTokenSecretKey},
		{Name: envEecIngestPort, Value: strconv.Itoa(int(collectorPort))},
		{Name: envExtensionsModuleExecPathName, Value: envExtensionsModuleExecPath},
		{Name: envDsInstallDirName, Value: envDsInstallDir},
		{Name: envK8sClusterId, Value: dk.Status.KubeSystemUUID},
		{Name: envK8sExtServiceUrl, Value: serviceAccountName},
		{Name: envDSTokenPath, Value: eecTokenMountPath + "/" + consts.OtelcTokenSecretKey},
		{Name: envHttpsCertPathPem, Value: envEecHttpsCertPathPem},
		{Name: envHttpsPrivKeyPathPem, Value: envEecHttpsPrivKeyPathPem},
	}

	if dk.Spec.ActiveGate.TlsSecretName != "" {
		containerEnvs = append(containerEnvs, corev1.EnvVar{Name: envActiveGateTrustedCertName, Value: envActiveGateTrustedCert})
	}

	if dk.Spec.Templates.ExtensionExecutionController.CustomConfig != "" {
		containerEnvs = append(containerEnvs, corev1.EnvVar{Name: envRuntimeConfigMountPath, Value: customConfigMountPath + "/" + runtimeConfigurationFilename})
	}

	return containerEnvs
}

func buildActiveGateServiceName(dk *dynakube.DynaKube) string {
	multiCap := capability.NewMultiCapability(dk)

	return capability.CalculateStatefulSetName(multiCap, dk.Name)
}

func buildContainerVolumeMounts(dk *dynakube.DynaKube) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      consts.ExtensionsTokensVolumeName,
			MountPath: eecTokenMountPath,
			ReadOnly:  true,
		},
		{
			Name:      logVolumeName,
			MountPath: logMountPath,
			ReadOnly:  false,
		},
		{
			Name:      runtimeVolumeName,
			MountPath: runtimeMountPath,
			ReadOnly:  false,
		},
		{
			Name:      configurationVolumeName,
			MountPath: configurationMountPath,
			ReadOnly:  false,
		},
		{
			Name:      httpsCertVolumeName,
			MountPath: httpsCertMountPath,
			ReadOnly:  true,
		},
	}

	if dk.Spec.Templates.ExtensionExecutionController.CustomConfig != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      customConfigVolumeName,
			MountPath: customConfigMountPath,
			ReadOnly:  true,
		})
	}

	if dk.Spec.ActiveGate.TlsSecretName != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      activeGateTrustedCertVolumeName,
			MountPath: activeGateTrustedCertMountPath,
			ReadOnly:  true,
		})
	}

	return volumeMounts
}

func setVolumes(dk *dynakube.DynaKube) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		tlsSecretName := extensionsControllerTlsSecretName

		if dk.Spec.Templates.ExtensionExecutionController.TlsRefName != "" {
			tlsSecretName = dk.Spec.Templates.ExtensionExecutionController.TlsRefName
		}

		mode := int32(420)
		o.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.Name + consts.SecretSuffix,
						DefaultMode: &mode,
					},
				},
			},
			{
				Name: logVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: configurationVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: httpsCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: tlsSecretName,
					},
				},
			},
		}

		if dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim == nil {
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: runtimeVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
		} else {
			// TODO: do we want to use statefulset.VolumeClaimTemplates
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: runtimeVolumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: runtimePersistentVolumeClaimName,
					},
				},
			})
		}

		if dk.Spec.Templates.ExtensionExecutionController.CustomConfig != "" {
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: customConfigVolumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: dk.Spec.Templates.ExtensionExecutionController.CustomConfig,
						},
					},
				},
			})
		}

		if dk.Spec.ActiveGate.TlsSecretName != "" {
			defaultMode := int32(420)
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: activeGateTrustedCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						DefaultMode: &defaultMode,
						SecretName:  dk.Spec.ActiveGate.TlsSecretName,
						Items: []corev1.KeyToPath{
							{
								Key:  activeGateTrustedCertSecretKeyPath,
								Path: activeGateTrustedCertSecretKeyPath,
							},
						},
					},
				},
			})
		}
	}
}
