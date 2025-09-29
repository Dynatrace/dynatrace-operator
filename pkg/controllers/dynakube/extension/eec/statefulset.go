package eec

import (
	"context"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	eecConsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/topology"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	containerName      = "extensions-controller"
	collectorPort      = int32(14599)
	serviceAccountName = "dynatrace-extensions-controller"

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
	eecTokenMountPath                  = "/var/lib/dynatrace/remotepluginmodule/secrets/tokens"
	customCertificateMountPath         = "/var/lib/dynatrace/remotepluginmodule/secrets/extensions"
	customCertificateVolumeName        = "extension-custom-certs"
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
	runtimeConfigurationFilename       = "runtimeConfiguration"
	serviceURLScheme                   = "https://"

	// misc
	logVolumeName = "log"

	userGroupID int64 = 1001
)

func (r *reconciler) createOrUpdateStatefulset(ctx context.Context) error {
	appLabels := buildAppLabels(r.dk.Name)

	templateAnnotations, err := r.buildTemplateAnnotations(ctx)
	if err != nil {
		return err
	}

	topologySpreadConstraints := topology.MaxOnePerNode(appLabels)
	if len(r.dk.Spec.Templates.ExtensionExecutionController.TopologySpreadConstraints) > 0 {
		topologySpreadConstraints = r.dk.Spec.Templates.ExtensionExecutionController.TopologySpreadConstraints
	}

	desiredSts, err := statefulset.Build(r.dk, r.dk.Extensions().GetExecutionControllerStatefulsetName(), buildContainer(r.dk),
		statefulset.SetReplicas(1),
		statefulset.SetPodManagementPolicy(appsv1.ParallelPodManagement),
		statefulset.SetAllLabels(appLabels.BuildLabels(), appLabels.BuildMatchLabels(), appLabels.BuildLabels(), r.dk.Spec.Templates.ExtensionExecutionController.Labels),
		statefulset.SetAllAnnotations(nil, templateAnnotations),
		statefulset.SetAffinity(buildAffinity()),
		statefulset.SetTolerations(r.dk.Spec.Templates.ExtensionExecutionController.Tolerations),
		statefulset.SetTopologySpreadConstraints(topologySpreadConstraints),
		statefulset.SetServiceAccount(serviceAccountName),
		statefulset.SetSecurityContext(buildPodSecurityContext(r.dk)),
		statefulset.SetRollingUpdateStrategyType(),
		setImagePullSecrets(r.dk.ImagePullSecretReferences()),
		setVolumes(r.dk),
		setPersistentVolumeClaim(r.dk),
	)
	if err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, err)

		return err
	}

	if err := hasher.AddAnnotation(desiredSts); err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, err)

		return err
	}

	_, err = statefulset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, desiredSts)
	if err != nil {
		log.Info("failed to create/update " + r.dk.Extensions().GetExecutionControllerStatefulsetName() + " statefulset")
		conditions.SetKubeAPIError(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, err)

		return err
	}

	conditions.SetStatefulSetCreated(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, desiredSts.Name)

	return nil
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

func buildAppLabels(dynakubeName string) *labels.AppLabels {
	return labels.NewAppLabels(labels.ExtensionComponentLabel, dynakubeName, labels.ExtensionComponentLabel, "")
}

func buildAffinity() corev1.Affinity {
	return node.Affinity()
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
	containerEnvs := []corev1.EnvVar{
		{Name: envTenantID, Value: dk.Status.ActiveGate.ConnectionInfo.TenantUUID},
		{Name: envServerURL, Value: buildActiveGateServiceName(dk) + "." + dk.Namespace + ":443"},
		{Name: envEecTokenPath, Value: eecTokenMountPath + "/" + eecConsts.TokenSecretKey},
		{Name: envEecIngestPort, Value: strconv.Itoa(int(collectorPort))},
		{Name: envExtensionsModuleExecPathName, Value: envExtensionsModuleExecPath},
		{Name: envDsInstallDirName, Value: envDsInstallDir},
		{Name: envK8sClusterID, Value: dk.Status.KubeSystemUUID},
		{Name: envK8sExtServiceURL, Value: serviceURLScheme + dk.Extensions().GetServiceNameFQDN()},
		{Name: envDSTokenPath, Value: eecTokenMountPath + "/" + consts.OtelcTokenSecretKey},
		{Name: envHTTPSCertPathPem, Value: envEecHTTPSCertPathPem},
		{Name: envHTTPSPrivKeyPathPem, Value: envEecHTTPSPrivKeyPathPem},
	}

	if dk.ActiveGate().HasCaCert() {
		containerEnvs = append(containerEnvs, corev1.EnvVar{Name: envActiveGateTrustedCertName, Value: envActiveGateTrustedCert})
	}

	if dk.Spec.Templates.ExtensionExecutionController.CustomConfig != "" {
		containerEnvs = append(containerEnvs, corev1.EnvVar{Name: envRuntimeConfigMountPath, Value: customConfigMountPath + "/" + runtimeConfigurationFilename})
	}

	if dk.Spec.Templates.ExtensionExecutionController.CustomExtensionCertificates != "" {
		containerEnvs = append(containerEnvs, corev1.EnvVar{Name: envCustomCertificateMountPath, Value: customCertificateMountPath})
	}

	return containerEnvs
}

func buildActiveGateServiceName(dk *dynakube.DynaKube) string {
	return capability.CalculateStatefulSetName(dk.Name)
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

	if dk.ActiveGate().HasCaCert() {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      activeGateTrustedCertVolumeName,
			MountPath: activeGateTrustedCertMountPath,
			ReadOnly:  true,
		})
	}

	if dk.Spec.Templates.ExtensionExecutionController.CustomExtensionCertificates != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      customCertificateVolumeName,
			MountPath: customCertificateMountPath,
			ReadOnly:  true,
		})
	}

	return volumeMounts
}

func setVolumes(dk *dynakube.DynaKube) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		mode := int32(420)
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
						SecretName: dk.Extensions().GetTLSSecretName(),
					},
				},
			},
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
