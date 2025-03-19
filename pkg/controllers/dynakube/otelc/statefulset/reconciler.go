package statefulset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/configuration"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/configmap"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/topology"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceAccountName                                  = "dynatrace-opentelemetry-collector"
	annotationTelemetryIngestSecretHash                 = api.InternalFlagPrefix + "telemetry-ingest-secret-hash"
	annotationTelemetryIngestConfigurationConfigMapHash = api.InternalFlagPrefix + "telemetry-ingest-config-hash"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

func NewReconciler(clt client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dk.IsExtensionsEnabled() || r.dk.TelemetryIngest().IsEnabled() {
		return r.createOrUpdateStatefulset(ctx)
	} else { // do cleanup or
		if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		sts, err := statefulset.Build(r.dk, r.dk.OtelCollectorStatefulsetName(), corev1.Container{})
		if err != nil {
			log.Error(err, "could not build "+r.dk.OtelCollectorStatefulsetName()+" during cleanup")

			return err
		}

		err = statefulset.Query(r.client, r.apiReader, log).Delete(ctx, sts)

		if err != nil {
			log.Error(err, "failed to clean up "+r.dk.OtelCollectorStatefulsetName()+" statufulset")

			return nil
		}

		return nil
	}
}

func (r *Reconciler) createOrUpdateStatefulset(ctx context.Context) error {
	if r.dk.TelemetryIngest().IsEnabled() {
		if !r.checkDataIngestTokenExists(ctx) {
			msg := "data ingest token is missing, but it's required for telemetery ingest"
			conditions.SetDataIngestTokenMissing(r.dk.Conditions(), dynakube.TokenConditionType, msg)

			log.Error(errors.New(msg), "could not create or update statefulset")

			return nil
		}
	}

	appLabels := buildAppLabels(r.dk.Name)

	templateAnnotations, err := r.buildTemplateAnnotations(ctx)
	if err != nil {
		return err
	}

	topologySpreadConstraints := topology.MaxOnePerNode(appLabels)
	if len(r.dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints) > 0 {
		topologySpreadConstraints = r.dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints
	}

	sts, err := statefulset.Build(r.dk, r.dk.OtelCollectorStatefulsetName(), getContainer(r.dk),
		statefulset.SetReplicas(getReplicas(r.dk)),
		statefulset.SetPodManagementPolicy(appsv1.ParallelPodManagement),
		statefulset.SetAllLabels(appLabels.BuildLabels(), appLabels.BuildMatchLabels(), appLabels.BuildLabels(), r.dk.Spec.Templates.OpenTelemetryCollector.Labels),
		statefulset.SetAllAnnotations(nil, templateAnnotations),
		statefulset.SetAffinity(buildAffinity()),
		statefulset.SetServiceAccount(serviceAccountName),
		statefulset.SetTolerations(r.dk.Spec.Templates.OpenTelemetryCollector.Tolerations),
		statefulset.SetTopologySpreadConstraints(topologySpreadConstraints),
		statefulset.SetSecurityContext(buildPodSecurityContext()),
		statefulset.SetRollingUpdateStrategyType(),
		setImagePullSecrets(r.dk.ImagePullSecretReferences()),
		setVolumes(r.dk),
	)

	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), conditionType, err)

		return err
	}

	if err := hasher.AddAnnotation(sts); err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), conditionType, err)

		return err
	}

	_, err = statefulset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, sts)
	if err != nil {
		log.Info("failed to create/update " + r.dk.OtelCollectorStatefulsetName() + " statefulset")
		conditions.SetKubeApiError(r.dk.Conditions(), conditionType, err)

		return err
	}

	conditions.SetStatefulSetCreated(r.dk.Conditions(), conditionType, sts.Name)

	return nil
}

func (r *Reconciler) buildTemplateAnnotations(ctx context.Context) (map[string]string, error) {
	templateAnnotations := map[string]string{}

	if r.dk.IsExtensionsEnabled() {
		if r.dk.Spec.Templates.OpenTelemetryCollector.Annotations != nil {
			templateAnnotations = r.dk.Spec.Templates.OpenTelemetryCollector.Annotations
		}

		tlsSecretHash, err := r.calculateSecretHash(ctx, r.dk.ExtensionsTLSSecretName())
		if err != nil {
			return nil, err
		}

		templateAnnotations[api.AnnotationExtensionsSecretHash] = tlsSecretHash
	}

	if r.dk.TelemetryIngest().IsEnabled() && r.dk.TelemetryIngest().Spec.TlsRefName != "" {
		tlsSecretHash, err := r.calculateSecretHash(ctx, r.dk.TelemetryIngest().Spec.TlsRefName)
		if err != nil {
			return nil, err
		}

		templateAnnotations[annotationTelemetryIngestSecretHash] = tlsSecretHash
	}

	if r.dk.TelemetryIngest().IsEnabled() {
		configConfigMapHash, err := r.calculateConfigMapHash(ctx, configuration.GetConfigMapName(r.dk.Name))
		if err != nil {
			return nil, err
		}

		templateAnnotations[annotationTelemetryIngestConfigurationConfigMapHash] = configConfigMapHash
	}

	return templateAnnotations, nil
}

func (r *Reconciler) calculateSecretHash(ctx context.Context, secretName string) (string, error) {
	query := k8ssecret.Query(r.client, r.client, log)

	tlsSecret, err := query.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: r.dk.Namespace,
	})
	if err != nil {
		return "", err
	}

	tlsSecretHash, err := hasher.GenerateHash(tlsSecret.Data)
	if err != nil {
		return "", err
	}

	return tlsSecretHash, nil
}

func (r *Reconciler) calculateConfigMapHash(ctx context.Context, configMapName string) (string, error) {
	query := configmap.Query(r.client, r.client, log)

	configConfigMap, err := query.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: r.dk.Namespace,
	})
	if err != nil {
		return "", err
	}

	configConfigMaptHash, err := hasher.GenerateHash(configConfigMap.Data)
	if err != nil {
		return "", err
	}

	return configConfigMaptHash, nil
}

func (r *Reconciler) checkDataIngestTokenExists(ctx context.Context) bool {
	tokenReader := token.NewReader(r.apiReader, r.dk)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		return false
	}

	return token.CheckForDataIngestToken(tokens)
}

func getReplicas(dk *dynakube.DynaKube) int32 {
	if dk.Spec.Templates.OpenTelemetryCollector.Replicas != nil {
		return *dk.Spec.Templates.OpenTelemetryCollector.Replicas
	}

	return defaultReplicas
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

func buildAppLabels(dkName string) *labels.AppLabels {
	// TODO: when version is available
	version := "0.0.0"

	return labels.NewAppLabels(labels.OtelCComponentLabel, dkName, labels.OtelCComponentLabel, version)
}

func buildAffinity() corev1.Affinity {
	return node.Affinity()
}

func setImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}
}
