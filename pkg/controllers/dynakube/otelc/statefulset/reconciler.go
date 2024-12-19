package statefulset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/topology"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceAccountName = "dynatrace-extensions-collector"
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
	if !r.dk.IsExtensionsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		sts, err := statefulset.Build(r.dk, r.dk.ExtensionsCollectorStatefulsetName(), corev1.Container{})
		if err != nil {
			log.Error(err, "could not build "+r.dk.ExtensionsCollectorStatefulsetName()+" during cleanup")

			return err
		}

		err = statefulset.Query(r.client, r.apiReader, log).Delete(ctx, sts)

		if err != nil {
			log.Error(err, "failed to clean up "+r.dk.ExtensionsCollectorStatefulsetName()+" statufulset")

			return nil
		}

		return nil
	}

	return r.createOrUpdateStatefulset(ctx)
}

func (r *Reconciler) createOrUpdateStatefulset(ctx context.Context) error {
	appLabels := buildAppLabels(r.dk.Name)

	templateAnnotations, err := r.buildTemplateAnnotations(ctx)
	if err != nil {
		return err
	}

	topologySpreadConstraints := topology.MaxOnePerNode(appLabels)
	if len(r.dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints) > 0 {
		topologySpreadConstraints = r.dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints
	}

	sts, err := statefulset.Build(r.dk, r.dk.ExtensionsCollectorStatefulsetName(), getContainer(r.dk),
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
		log.Info("failed to create/update " + r.dk.ExtensionsCollectorStatefulsetName() + " statefulset")
		conditions.SetKubeApiError(r.dk.Conditions(), conditionType, err)

		return err
	}

	conditions.SetStatefulSetCreated(r.dk.Conditions(), conditionType, sts.Name)

	return nil
}

func (r *Reconciler) buildTemplateAnnotations(ctx context.Context) (map[string]string, error) {
	templateAnnotations := map[string]string{}

	if r.dk.Spec.Templates.OpenTelemetryCollector.Annotations != nil {
		templateAnnotations = r.dk.Spec.Templates.OpenTelemetryCollector.Annotations
	}

	query := k8ssecret.Query(r.client, r.client, log)

	tlsSecret, err := query.Get(ctx, types.NamespacedName{
		Name:      r.dk.ExtensionsTLSSecretName(),
		Namespace: r.dk.Namespace,
	})
	if err != nil {
		return nil, err
	}

	tlsSecretHash, err := hasher.GenerateHash(tlsSecret.Data)
	if err != nil {
		return nil, err
	}

	templateAnnotations[api.AnnotationSecretHash] = tlsSecretHash

	return templateAnnotations, nil
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

	return labels.NewAppLabels(labels.CollectorComponentLabel, dkName, labels.CollectorComponentLabel, version)
}

func buildAffinity() corev1.Affinity {
	return node.Affinity()
}

func setImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}
}
