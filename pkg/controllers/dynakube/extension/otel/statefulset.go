package otel

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/hash"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/utils"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	serviceAccountName       = "dynatrace-extensions-collector"
	containerName            = "collector"
	tokenSecretKey           = "otelc.token"
	caCertsVolumeName        = "cacerts"
	defaultImageRepo         = "public.ecr.aws/dynatrace/dynatrace-otel-collector"
	defaultImageTag          = "0.7.0"
	defaultOLTPgrpcPort      = "10001"
	defaultOLTPhttpPort      = "10002"
	defaultPodNamePrefix     = "extensions-collector"
	defaultReplicas          = 1
	envShards                = "SHARDS"
	envShardId               = "SHARD_ID"
	envPodNamePrefix         = "POD_NAME_PREFIX"
	envPodName               = "POD_NAME"
	envOTLPgrpcPort          = "OTLP_GRPC_PORT"
	envOTLPhttpPort          = "OTLP_HTTP_PORT"
	envOTLPtoken             = "OTLP_TOKEN"
	envTrustedCAs            = "TRUSTED_CAS"
	trustedCAVolumeMountPath = "/tls/custom/cacerts"
	trustedCAVolumePath      = trustedCAVolumeMountPath + "/certs"
)

func (r *reconciler) createOrUpdateStatefulset(ctx context.Context) error {
	appLabels := buildAppLabels(r.dk.Name)
	sts, err := statefulset.Build(r.dk, dynakube.ExtensionsCollectorStatefulsetName, buildContainer(r.dk),
		statefulset.SetReplicas(getReplicas(r.dk)),
		statefulset.SetPodManagementPolicy(appsv1.ParallelPodManagement),
		statefulset.SetAllLabels(appLabels.BuildLabels(), appLabels.BuildMatchLabels(), appLabels.BuildLabels(), r.dk.Spec.Templates.OpenTelemetryCollector.Labels),
		statefulset.SetAllAnnotations(nil, r.dk.Spec.Templates.OpenTelemetryCollector.Annotations),
		statefulset.SetAffinity(buildAffinity()),
		statefulset.SetServiceAccount(serviceAccountName),
		statefulset.SetTolerations(r.dk.Spec.Templates.OpenTelemetryCollector.Tolerations),
		statefulset.SetTopologySpreadConstraints(utils.BuildTopologySpreadConstraints(r.dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints, appLabels)),
		statefulset.SetSecurityContext(buildPodSecurityContext()),
		statefulset.SetUpdateStrategy(utils.BuildUpdateStrategy()),
		setTlsRef(r.dk.Spec.Templates.OpenTelemetryCollector.TlsRefName),
		setImagePullSecrets(r.dk.ImagePullSecretReferences()),
		setVolumes(r.dk),
	)

	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), otelControllerStatefulSetConditionType, err)

		return err
	}

	if err := hash.SetHash(sts); err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), otelControllerStatefulSetConditionType, err)

		return err
	}

	_, err = statefulset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, sts)
	if err != nil {
		log.Info("failed to create/update " + dynakube.ExtensionsCollectorStatefulsetName + " statefulset")
		conditions.SetKubeApiError(r.dk.Conditions(), otelControllerStatefulSetConditionType, err)

		return err
	}

	conditions.SetStatefulSetCreated(r.dk.Conditions(), otelControllerStatefulSetConditionType, sts.Name)

	return nil
}

func getReplicas(dk *dynakube.DynaKube) int32 {
	if dk.Spec.Templates.OpenTelemetryCollector.Replicas != nil {
		return *dk.Spec.Templates.OpenTelemetryCollector.Replicas
	}

	return defaultReplicas
}

func buildContainer(dk *dynakube.DynaKube) corev1.Container {
	imageRepo := dk.Spec.Templates.OpenTelemetryCollector.ImageRef.Repository
	imageTag := dk.Spec.Templates.OpenTelemetryCollector.ImageRef.Tag

	if imageRepo == "" {
		imageRepo = defaultImageRepo
	}

	if imageTag == "" {
		imageTag = defaultImageTag
	}

	return corev1.Container{
		Name:            containerName,
		Image:           imageRepo + ":" + imageTag,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: buildSecurityContext(),
		Env:             buildContainerEnvs(dk),
		Resources:       dk.Spec.Templates.OpenTelemetryCollector.Resources,
		Args:            []string{fmt.Sprintf("--config=eec://%s.%s.svc.cluster.local:%d#refresh-interval=5s&insecure=true", dk.Name+consts.ExtensionsControllerSuffix, dk.Namespace, consts.ExtensionsCollectorComPort)},
		VolumeMounts:    buildContainerVolumeMounts(dk),
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
	envs := []corev1.EnvVar{
		{Name: envShards, Value: strconv.Itoa(int(getReplicas(dk)))},
		{Name: envPodNamePrefix, Value: defaultPodNamePrefix},
		{Name: envPodName, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['statefulset.kubernetes.io/pod-name']",
			},
		},
		},
		{Name: envShardId, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['app.kubernetes.io/pod-index']",
			},
		},
		},
		{Name: envOTLPgrpcPort, Value: defaultOLTPgrpcPort},
		{Name: envOTLPhttpPort, Value: defaultOLTPhttpPort},
		{Name: envOTLPtoken, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dk.Name + consts.SecretSuffix},
				Key:                  tokenSecretKey,
			},
		},
		},
	}
	if dk.Spec.TrustedCAs != "" {
		envs = append(envs, corev1.EnvVar{Name: envTrustedCAs, Value: trustedCAVolumePath})
	}

	return envs
}

func buildAppLabels(dkName string) *labels.AppLabels {
	// TODO: when version is available
	version := "0.0.0"

	return labels.NewAppLabels(labels.CollectorComponentLabel, dkName, labels.CollectorComponentLabel, version)
}

func buildAffinity() corev1.Affinity {
	// TODO: implement new attributes in CR dk.Spec.Templates.OpenTelemetryCollector.Affinity
	// otherwise to use defaults ones
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
		// TODO:
	}
}

func setImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}
}

func setVolumes(dk *dynakube.DynaKube) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		if dk.Spec.TrustedCAs != "" {
			o.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: caCertsVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dk.Spec.TrustedCAs,
						},
					},
				},
			}
		}
	}
}

func buildContainerVolumeMounts(dk *dynakube.DynaKube) []corev1.VolumeMount {
	if dk.Spec.TrustedCAs != "" {
		return []corev1.VolumeMount{
			{
				Name:      caCertsVolumeName,
				MountPath: trustedCAVolumeMountPath,
				ReadOnly:  true,
			},
		}
	}

	return []corev1.VolumeMount{}
}
