package eec

import (
	"context"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	eecConsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8saffinity"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8stopology"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	containerName      = "extension-controller"
	serviceAccountName = "dynatrace-extension-controller"

	// Env variable names
	envTenantID                     = "TenantId"
	envServerURL                    = "ServerUrl"
	envEecTokenPath                 = "EecTokenPath"
	envEecIngestPort                = "EecIngestPort"
	envExtensionsModuleExecPathName = "ExtensionsModuleExecPath"
	envDsInstallDirName             = "DsInstallDir"
	envK8sClusterID                 = "K8sClusterUID"
	envActiveGateTrustedCertName    = "ActiveGateTrustedCert"
	envK8sExtServiceURL             = "K8sExtServiceUrl"
	envHTTPSCertPathPem             = "DsHttpsCertPathPem"
	envHTTPSPrivKeyPathPem          = "DsHttpsPrivKeyPathPem"
	envDSTokenPath                  = "DSTokenPath"
	envRuntimeConfigMountPath       = "RuntimeConfigMountPath"
	envCustomCertificateMountPath   = "ExtensionCustomCertificateMountPath"
	// Env variable values
	envExtensionsModuleExecPath = "/opt/dynatrace/remotepluginmodule/agent/lib64/extensionsmodule"
	envDsInstallDir             = "/opt/dynatrace/remotepluginmodule/agent/datasources"
	envActiveGateTrustedCert    = activeGateTrustedCertMountPath + "/" + activeGateTrustedCertSecretKeyPath
	envEecHTTPSCertPathPem      = httpsCertMountPath + "/" + consts.TLSCrtDataName
	envEecHTTPSPrivKeyPathPem   = httpsCertMountPath + "/" + consts.TLSKeyDataName
	// Volume names and paths
	eecTokenMountPath                  = "/secrets/tokens"
	customCertificateMountPath         = "/secrets/extensions"
	customCertificateVolumeName        = "extension-custom-certs"
	runtimeVolumeName                  = "agent-runtime"
	runtimeMountPath                   = "/var/lib/dynatrace/remotepluginmodule"
	customConfigVolumeName             = "custom-config"
	customConfigMountPath              = "/secrets/config"
	activeGateTrustedCertVolumeName    = "server-certs"
	activeGateTrustedCertMountPath     = "/secrets/ag"
	activeGateTrustedCertSecretKeyPath = "server.crt"
	httpsCertVolumeName                = "https-certs"
	httpsCertMountPath                 = "/secrets/https"
	runtimeConfigurationFilename       = "runtimeConfiguration"
	serviceURLScheme                   = "https://"

	legacyConfigurationVolumeName = "runtime-configuration"
	legacyConfigurationMountPath  = "/var/lib/dynatrace/remotepluginmodule/agent/conf"
	legacyRuntimeMountPath        = "/var/lib/dynatrace/remotepluginmodule/agent/runtime"
	legacyLogVolumeName           = "log"
	legacyLogMountPath            = "/var/lib/dynatrace/remotepluginmodule/log"

	userGroupID int64 = 1001
)

func useLegacyMounts(dk *dynakube.DynaKube) bool {
	return exp.NewFlags(dk.Annotations).UseEECLegacyMounts()
}

func (r *reconciler) createOrUpdateStatefulset(ctx context.Context) error {
	appLabels := buildAppLabels(r.dk.Name)

	templateAnnotations, err := r.buildTemplateAnnotations(ctx)
	if err != nil {
		return err
	}

	topologySpreadConstraints := k8stopology.MaxOnePerNode(appLabels)
	if len(r.dk.Spec.Templates.ExtensionExecutionController.TopologySpreadConstraints) > 0 {
		topologySpreadConstraints = r.dk.Spec.Templates.ExtensionExecutionController.TopologySpreadConstraints
	}

	desiredSts, err := k8sstatefulset.Build(r.dk, r.dk.Extensions().GetExecutionControllerStatefulsetName(), buildContainer(r.dk),
		k8sstatefulset.SetReplicas(1),
		k8sstatefulset.SetPodManagementPolicy(appsv1.ParallelPodManagement),
		k8sstatefulset.SetAllLabels(appLabels.BuildLabels(), appLabels.BuildMatchLabels(), appLabels.BuildLabels(), r.dk.Spec.Templates.ExtensionExecutionController.Labels),
		k8sstatefulset.SetAllAnnotations(nil, templateAnnotations),
		k8sstatefulset.SetAffinity(buildAffinity()),
		k8sstatefulset.SetTolerations(r.dk.Spec.Templates.ExtensionExecutionController.Tolerations),
		k8sstatefulset.SetTopologySpreadConstraints(topologySpreadConstraints),
		k8sstatefulset.SetServiceAccount(serviceAccountName),
		k8sstatefulset.SetSecurityContext(buildPodSecurityContext(r.dk)),
		k8sstatefulset.SetRollingUpdateStrategyType(),
		setImagePullSecrets(r.dk.ImagePullSecretReferences()),
		setVolumes(r.dk),
		setPersistentVolumeClaim(r.dk),
	)
	if err != nil {
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), extensionControllerStatefulSetConditionType, err)

		return err
	}

	_, err = k8sstatefulset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, desiredSts)
	if err != nil {
		log.Info("failed to create/update " + r.dk.Extensions().GetExecutionControllerStatefulsetName() + " statefulset")
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), extensionControllerStatefulSetConditionType, err)

		return err
	}

	k8sconditions.SetStatefulSetCreated(r.dk.Conditions(), extensionControllerStatefulSetConditionType, desiredSts.Name)

	return nil
}

// TODO: Remove as part of DAQ-18375
func (r *reconciler) deleteLegacyStatefulset(ctx context.Context) {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.dk.Name + "-extensions-controller",
			Namespace: r.dk.Namespace,
		},
	}

	_ = r.client.Delete(ctx, sts)
}

func (r *reconciler) buildTemplateAnnotations(ctx context.Context) (map[string]string, error) {
	templateAnnotations := map[string]string{}

	if r.dk.Spec.Templates.ExtensionExecutionController.Annotations != nil {
		templateAnnotations = r.dk.Spec.Templates.ExtensionExecutionController.Annotations
	}

	secrets := k8ssecret.Query(r.client, r.client, log)

	tlsSecret, err := secrets.Get(ctx, types.NamespacedName{
		Name:      r.dk.Extensions().GetTLSSecretName(),
		Namespace: r.dk.Namespace,
	})
	if err != nil {
		return nil, err
	}

	tlsSecretHash, err := hasher.GenerateHash(tlsSecret.Data)
	if err != nil {
		return nil, err
	}

	templateAnnotations[api.AnnotationExtensionsSecretHash] = tlsSecretHash

	return templateAnnotations, nil
}

func buildAppLabels(dynakubeName string) *k8slabel.AppLabels {
	return k8slabel.NewAppLabels(k8slabel.ExtensionComponentLabel, dynakubeName, k8slabel.ExtensionComponentLabel, "")
}

func buildAffinity() corev1.Affinity {
	return k8saffinity.NewMultiArchNodeAffinity()
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
					Port:   intstr.IntOrString{IntVal: consts.ExtensionsDatasourceTargetPort},
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
				Name:          consts.ExtensionsDatasourceTargetPortName,
				ContainerPort: consts.ExtensionsDatasourceTargetPort,
			},
		},
		Env:          buildContainerEnvs(dk),
		Resources:    dk.Spec.Templates.ExtensionExecutionController.Resources,
		VolumeMounts: buildContainerVolumeMounts(dk),
	}
}

func buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		Privileged:               ptr.To(false),
		RunAsUser:                ptr.To(userGroupID),
		RunAsGroup:               ptr.To(userGroupID),
		RunAsNonRoot:             ptr.To(true),
		ReadOnlyRootFilesystem:   ptr.To(true),
		AllowPrivilegeEscalation: ptr.To(false),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func buildPodSecurityContext(dk *dynakube.DynaKube) *corev1.PodSecurityContext {
	podSecurityContext := &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	if !dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume {
		podSecurityContext.FSGroup = ptr.To(userGroupID)
	}

	return podSecurityContext
}

func buildContainerEnvs(dk *dynakube.DynaKube) []corev1.EnvVar {
	var prefix string
	if useLegacyMounts(dk) {
		prefix = runtimeMountPath
	}

	containerEnvs := []corev1.EnvVar{
		{Name: envTenantID, Value: dk.Status.ActiveGate.ConnectionInfo.TenantUUID},
		{Name: envServerURL, Value: buildActiveGateServiceName(dk) + "." + dk.Namespace + ":443"},
		{Name: envEecTokenPath, Value: prefix + eecTokenMountPath + "/" + eecConsts.TokenSecretKey},
		{Name: envEecIngestPort, Value: strconv.Itoa(consts.ExtensionsDatasourceTargetPort)},
		{Name: envExtensionsModuleExecPathName, Value: envExtensionsModuleExecPath},
		{Name: envDsInstallDirName, Value: envDsInstallDir},
		{Name: envK8sClusterID, Value: dk.Status.KubeSystemUUID},
		{Name: envK8sExtServiceURL, Value: serviceURLScheme + dk.Extensions().GetServiceNameFQDN()},
		{Name: envDSTokenPath, Value: prefix + eecTokenMountPath + "/" + consts.DatasourceTokenSecretKey},
		{Name: envHTTPSCertPathPem, Value: prefix + envEecHTTPSCertPathPem},
		{Name: envHTTPSPrivKeyPathPem, Value: prefix + envEecHTTPSPrivKeyPathPem},
	}

	if dk.ActiveGate().HasCaCert() {
		containerEnvs = append(containerEnvs, corev1.EnvVar{Name: envActiveGateTrustedCertName, Value: prefix + envActiveGateTrustedCert})
	}

	if dk.Spec.Templates.ExtensionExecutionController.CustomConfig != "" {
		containerEnvs = append(containerEnvs, corev1.EnvVar{Name: envRuntimeConfigMountPath, Value: prefix + customConfigMountPath + "/" + runtimeConfigurationFilename})
	}

	if dk.Spec.Templates.ExtensionExecutionController.CustomExtensionCertificates != "" {
		containerEnvs = append(containerEnvs, corev1.EnvVar{Name: envCustomCertificateMountPath, Value: prefix + customCertificateMountPath})
	}

	return containerEnvs
}

func buildActiveGateServiceName(dk *dynakube.DynaKube) string {
	return capability.CalculateStatefulSetName(dk.Name)
}

func buildContainerVolumeMounts(dk *dynakube.DynaKube) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	if useLegacyMounts(dk) {
		volumeMounts = []corev1.VolumeMount{
			{
				Name:      consts.ExtensionsTokensVolumeName,
				MountPath: runtimeMountPath + eecTokenMountPath,
				ReadOnly:  true,
			},
			{
				Name:      legacyLogVolumeName,
				MountPath: legacyLogMountPath,
				ReadOnly:  false,
			},
			{
				Name:      runtimeVolumeName,
				MountPath: legacyRuntimeMountPath,
				ReadOnly:  false,
			},
			{
				Name:      legacyConfigurationVolumeName,
				MountPath: legacyConfigurationMountPath,
				ReadOnly:  false,
			},
			{
				Name:      httpsCertVolumeName,
				MountPath: runtimeMountPath + httpsCertMountPath,
				ReadOnly:  true,
			},
		}
	} else {
		volumeMounts = []corev1.VolumeMount{
			{
				Name:      consts.ExtensionsTokensVolumeName,
				MountPath: eecTokenMountPath,
				ReadOnly:  true,
			},
			{
				Name:      runtimeVolumeName,
				MountPath: runtimeMountPath,
				ReadOnly:  false,
			},
			{
				Name:      httpsCertVolumeName,
				MountPath: httpsCertMountPath,
				ReadOnly:  true,
			},
		}
	}

	if dk.Spec.Templates.ExtensionExecutionController.CustomConfig != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name: customConfigVolumeName,
			MountPath: func() string {
				var prefix string
				if useLegacyMounts(dk) {
					prefix = runtimeMountPath
				}

				return prefix + customConfigMountPath
			}(),
			ReadOnly: true,
		})
	}

	if dk.ActiveGate().HasCaCert() {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name: activeGateTrustedCertVolumeName,
			MountPath: func() string {
				var prefix string
				if useLegacyMounts(dk) {
					prefix = runtimeMountPath
				}

				return prefix + activeGateTrustedCertMountPath
			}(),
			ReadOnly: true,
		})
	}

	if dk.Spec.Templates.ExtensionExecutionController.CustomExtensionCertificates != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name: customCertificateVolumeName,
			MountPath: func() string {
				var prefix string
				if useLegacyMounts(dk) {
					prefix = runtimeMountPath
				}

				return prefix + customCertificateMountPath
			}(),
			ReadOnly: true,
		})
	}

	return volumeMounts
}

func setVolumes(dk *dynakube.DynaKube) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		mode := int32(420)
		if useLegacyMounts(dk) {
			o.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: consts.ExtensionsTokensVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  dk.Extensions().GetTokenSecretName(),
							DefaultMode: &mode,
						},
					},
				},
				{
					Name: legacyLogVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: legacyConfigurationVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: httpsCertVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dk.Extensions().GetTLSSecretName(),
						},
					},
				},
			}
		} else {
			o.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: consts.ExtensionsTokensVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  dk.Extensions().GetTokenSecretName(),
							DefaultMode: &mode,
						},
					},
				},
				{
					Name: httpsCertVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dk.Extensions().GetTLSSecretName(),
						},
					},
				},
			}
		}

		if dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume {
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: runtimeVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
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

		if dk.ActiveGate().HasCaCert() {
			defaultMode := int32(420)
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: activeGateTrustedCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						DefaultMode: &defaultMode,
						SecretName:  dk.ActiveGate().GetTLSSecretName(),
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

		if dk.Spec.Templates.ExtensionExecutionController.CustomExtensionCertificates != "" {
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: customCertificateVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dk.Spec.Templates.ExtensionExecutionController.CustomExtensionCertificates,
					},
				},
			})
		}
	}
}

func setPersistentVolumeClaim(dk *dynakube.DynaKube) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		if !dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume {
			if dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim == nil {
				o.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: runtimeVolumeName,
						},
						Spec: defaultPVCSpec(),
					},
				}
			} else {
				o.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: runtimeVolumeName,
						},
						Spec: *dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim,
					},
				}
			}

			o.Spec.PersistentVolumeClaimRetentionPolicy = &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
				WhenScaled:  appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
			}
		}
	}
}

func defaultPVCSpec() corev1.PersistentVolumeClaimSpec {
	return corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}
}
