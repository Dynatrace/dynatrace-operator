package statefulset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/configuration"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8saffinity"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8stopology"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sconfigmap"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceAccountName                                  = "dynatrace-otel-collector"
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
	if r.dk.Extensions().IsPrometheusEnabled() || r.dk.TelemetryIngest().IsEnabled() {
		return r.createOrUpdateStatefulset(ctx)
	} else { // do cleanup or
		if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		sts, err := k8sstatefulset.Build(r.dk, r.dk.OtelCollectorStatefulsetName(), corev1.Container{})
		if err != nil {
			log.Error(err, "could not build "+r.dk.OtelCollectorStatefulsetName()+" during cleanup")

			return err
		}

		err = k8sstatefulset.Query(r.client, r.apiReader, log).Delete(ctx, sts)
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

	topologySpreadConstraints := k8stopology.MaxOnePerNode(appLabels)
	if len(r.dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints) > 0 {
		topologySpreadConstraints = r.dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints
	}

	sts, err := k8sstatefulset.Build(r.dk, r.dk.OtelCollectorStatefulsetName(), getContainer(r.dk),
		k8sstatefulset.SetReplicas(getReplicas(r.dk)),
		k8sstatefulset.SetPodManagementPolicy(appsv1.ParallelPodManagement),
		k8sstatefulset.SetAllLabels(appLabels.BuildLabels(), appLabels.BuildMatchLabels(), appLabels.BuildLabels(), r.dk.Spec.Templates.OpenTelemetryCollector.Labels),
		k8sstatefulset.SetAllAnnotations(nil, templateAnnotations),
		k8sstatefulset.SetAffinity(buildAffinity()),
		k8sstatefulset.SetServiceAccount(serviceAccountName),
		k8sstatefulset.SetTolerations(r.dk.Spec.Templates.OpenTelemetryCollector.Tolerations),
		k8sstatefulset.SetTopologySpreadConstraints(topologySpreadConstraints),
		k8sstatefulset.SetSecurityContext(buildPodSecurityContext()),
		k8sstatefulset.SetRollingUpdateStrategyType(),
		setImagePullSecrets(r.dk.ImagePullSecretReferences()),
		setVolumes(r.dk),
	)
	if err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	_, err = k8sstatefulset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, sts)
	if err != nil {
		log.Info("failed to create/update " + r.dk.OtelCollectorStatefulsetName() + " statefulset")
		conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	conditions.SetStatefulSetCreated(r.dk.Conditions(), conditionType, sts.Name)

	return nil
}

func (r *Reconciler) buildTemplateAnnotations(ctx context.Context) (map[string]string, error) {
	templateAnnotations := map[string]string{}

	if r.dk.Extensions().IsPrometheusEnabled() {
		if r.dk.Spec.Templates.OpenTelemetryCollector.Annotations != nil {
			templateAnnotations = r.dk.Spec.Templates.OpenTelemetryCollector.Annotations
		}

		tlsSecretHash, err := r.calculateSecretHash(ctx, r.dk.Extensions().GetTLSSecretName())
		if err != nil {
			return nil, err
		}

		templateAnnotations[api.AnnotationExtensionsSecretHash] = tlsSecretHash
	}

	if r.dk.TelemetryIngest().IsEnabled() && r.dk.TelemetryIngest().TLSRefName != "" {
		tlsSecretHash, err := r.calculateSecretHash(ctx, r.dk.TelemetryIngest().TLSRefName)
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
	secrets := k8ssecret.Query(r.client, r.client, log)

	tlsSecret, err := secrets.Get(ctx, types.NamespacedName{
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
	query := k8sconfigmap.Query(r.client, r.client, log)

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

func buildAppLabels(dkName string) *k8slabel.AppLabels {
	return k8slabel.NewAppLabels(k8slabel.OtelCComponentLabel, dkName, k8slabel.OtelCComponentLabel, "")
}

func buildAffinity() corev1.Affinity {
	return k8saffinity.NewMultiArchNodeAffinity()
}

func setImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}
}
