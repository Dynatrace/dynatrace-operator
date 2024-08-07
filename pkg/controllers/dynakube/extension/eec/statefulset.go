package eec

import (
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/hash"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/utils"
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
	statefulsetName                  = "dynatrace-extensions-controller"
	runtimePersistentVolumeClaimName = statefulsetName + "-runtime"
	containerName                    = "extensions-controller"
	collectorPort                    = int32(14599)
	serviceAccountName               = "dynatrace-extensions-controller"

	envTenantId                     = "TenantId"
	envServerUrl                    = "ServerUrl"
	envEecTokenPath                 = "EecTokenPath"
	envEecIngestPort                = "EecIngestPort"
	envExtensionsConfPathName       = "ExtensionsConfPath"
	envExtensionsConfPath           = "/opt/dynatrace/remotepluginmodule/agent/conf/extensions.conf"
	envExtensionsModuleExecPathName = "ExtensionsModuleExecPath"
	envExtensionsModuleExecPath     = "/opt/dynatrace/remotepluginmodule/agent/lib64/extensionsmodule"
	envDsInstallDirName             = "DsInstallDir"
	envDsInstallDir                 = "/opt/dynatrace/remotepluginmodule/agent/datasources"
	envK8sClusterId                 = "K8sClusterId"

	tokensVolumeName        = "tokens"
	eecTokenMountPath       = "/var/lib/dynatrace/remotepluginmodule/secrets/tokens"
	eecFile                 = "eec.token"
	logVolumeName           = "log"
	logMountPath            = "/var/lib/dynatrace/remotepluginmodule/log"
	runtimeVolumeName       = "agent-runtime"
	runtimeMountPath        = "/var/lib/dynatrace/remotepluginmodule/agent/runtime"
	configurationVolumeName = "runtime-configuration"
	configurationMountPath  = "/var/lib/dynatrace/remotepluginmodule/agent/conf/runtime"
)

func (r *reconciler) createOrUpdateStatefulset(ctx context.Context) error {
	appLabels := buildAppLabels(r.dk.Name)
	desiredSts, err := statefulset.Build(r.dk, statefulsetName, buildContainer(r.dk),
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
		setTlsRef(r.dk.Spec.Templates.ExtensionExecutionController.TlsRefName),
		setImagePullSecrets(r.dk.ImagePullSecretReferences()),
		setVolumes(r.dk.Name, r.dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim),
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
		log.Info("failed to create/update " + statefulsetName + " statefulset")
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

func setTlsRef(tlsRefName string) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		// TODO: EEC image is ready
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
					Scheme: "HTTP",
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
		VolumeMounts: buildContainerVolumeMounts(),
	}
}

func buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
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
	return []corev1.EnvVar{
		{Name: envTenantId, Value: dk.Status.ActiveGate.ConnectionInfoStatus.TenantUUID},
		{Name: envServerUrl, Value: buildActiveGateServiceName(dk) + "." + dk.Namespace + ".svc.cluster.local:443"},
		{Name: envEecTokenPath, Value: eecTokenMountPath + "/" + eecFile},
		{Name: envEecIngestPort, Value: strconv.Itoa(int(collectorPort))},
		{Name: envExtensionsConfPathName, Value: envExtensionsConfPath},
		{Name: envExtensionsModuleExecPathName, Value: envExtensionsModuleExecPath},
		{Name: envDsInstallDirName, Value: envDsInstallDir},
		{Name: envK8sClusterId, Value: dk.Status.KubeSystemUUID},
	}
}

func buildActiveGateServiceName(dk *dynakube.DynaKube) string {
	multiCap := capability.NewMultiCapability(dk)

	return capability.CalculateStatefulSetName(multiCap, dk.Name)
}

func buildContainerVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      tokensVolumeName,
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
			ReadOnly:  true,
		},
	}
}

func setVolumes(dynakubeName string, claim *corev1.PersistentVolumeClaimSpec) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		mode := int32(420)
		o.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: tokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dynakubeName + consts.SecretSuffix,
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
				// TODO: is a configMap.name: eec-runtime-configuration needed?
				Name: configurationVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}

		if claim == nil {
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
	}
}
