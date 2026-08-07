// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package injection

import (
	"context"
	goerrors "errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	dtoneagent "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	dtsettings "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	dtversion "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/version"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/metadata/rules"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/otlp/exporterconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type istioReconciler interface {
	ReconcileCodeModules(ctx context.Context, dk *dynakube.DynaKube) error
}

type versionReconciler interface {
	ReconcileCodeModules(ctx context.Context, dk *dynakube.DynaKube, imageClient dtimage.Client, versionClient dtversion.Client) error
}

type connectionInfoReconciler interface {
	Reconcile(ctx context.Context, oaClient dtoneagent.Client, dk *dynakube.DynaKube) error
}

type enrichmentRulesReconciler interface {
	Reconcile(ctx context.Context, dtClient dtsettings.Client, dk *dynakube.DynaKube) error
}

type secretGenerator interface {
	GenerateForDynakube(ctx context.Context, dk *dynakube.DynaKube, namespaces []corev1.Namespace) error
}

type Reconciler struct {
	client                    client.Client
	apiReader                 client.Reader
	istioReconciler           istioReconciler
	versionReconciler         versionReconciler
	connectionInfoReconciler  connectionInfoReconciler
	enrichmentRulesReconciler enrichmentRulesReconciler
}

var newBootstrapSecretGenerator = newDefaultBootstrapperSecretGenerator

func newDefaultBootstrapperSecretGenerator(client client.Client, apiReader client.Reader, dtClient dtoneagent.Client) secretGenerator {
	return bootstrapperconfig.NewSecretGenerator(client, apiReader, dtClient)
}

var newExporterSecretGenerator = newDefaultExporterSecretGenerator

func newDefaultExporterSecretGenerator(client client.Client, apiReader client.Reader) secretGenerator {
	return exporterconfig.NewSecretGenerator(client, apiReader)
}

func NewReconciler(
	client client.Client,
	apiReader client.Reader,
) *Reconciler {
	return &Reconciler{
		client:                    client,
		apiReader:                 apiReader,
		istioReconciler:           istio.NewReconciler(client, apiReader),
		versionReconciler:         version.NewReconciler(apiReader),
		connectionInfoReconciler:  oaconnectioninfo.NewReconciler(client, apiReader),
		enrichmentRulesReconciler: rules.NewReconciler(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dtClient *dynatrace.Client, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "injection")

	err := r.reconcileSubReconcilers(ctx, dtClient, dk)
	if err != nil {
		return err
	}

	var setupErrors []error

	if !dk.OneAgent().IsAppInjectionNeeded() && !dk.MetadataEnrichment().IsEnabled() && !dk.OTLPExporterConfiguration().IsEnabled() {
		defer r.unmap(ctx, dk)
	} else {
		dkMapper := r.createDynakubeMapper(ctx, dk)

		if err := dkMapper.MapFromDynakube(); err != nil {
			log.Info("update of a map of namespaces failed")

			setupErrors = append(setupErrors, err)
		}
	}

	namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, dk.Name)
	if err != nil {
		return err
	}

	if err := r.setupInitSecret(ctx, dtClient, namespaces, dk); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if err := r.setupOTLPSecret(ctx, namespaces, dk); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if len(setupErrors) > 0 {
		return goerrors.Join(setupErrors...)
	}

	log.Info("app injection reconciled")

	return nil
}

func (r *Reconciler) reconcileSubReconcilers(ctx context.Context, dtClient *dynatrace.Client, dk *dynakube.DynaKube) error {
	var setupErrors []error
	if err := r.setupOneAgentInjection(ctx, dk, dtClient); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if err := r.setupEnrichmentInjection(ctx, dk, dtClient.Settings); err != nil {
		setupErrors = append(setupErrors, err)
	}

	return goerrors.Join(setupErrors...)
}

func (r *Reconciler) setupOTLPSecret(ctx context.Context, namespaces []corev1.Namespace, dk *dynakube.DynaKube) error {
	if dk.OTLPExporterConfiguration().IsEnabled() {
		namespaces := filterNamespaces(ctx, namespaces, &dk.Spec.OTLPExporterConfiguration.NamespaceSelector)

		if err := r.generateOTLPSecret(ctx, namespaces, dk); err != nil {
			return err
		}

		setOTLPExporterConfigurationCondition(dk.Conditions())
	} else {
		r.cleanupOTLPSecret(ctx, namespaces, dk)
	}

	return nil
}

func (r *Reconciler) setupInitSecret(ctx context.Context, dtClient *dynatrace.Client, namespaces []corev1.Namespace, dk *dynakube.DynaKube) error {
	if bootstrapperconfig.NeedsPGC(dk) || dk.MetadataEnrichment().IsEnabled() {
		namespaces := filterNamespaces(ctx, namespaces, dk.OneAgent().GetNamespaceSelector(), dk.MetadataEnrichment().GetNamespaceSelector())

		return r.generateInitSecret(ctx, dtClient, namespaces, dk)
	}

	r.cleanupInitSecret(ctx, namespaces, dk)

	return nil
}

func (r *Reconciler) unmap(ctx context.Context, dk *dynakube.DynaKube) {
	log := logd.FromContext(ctx)

	namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, dk.Name)
	if err != nil {
		log.Error(err, "failed to list namespaces for dynakube")
	}

	dkMapper := r.createDynakubeMapper(ctx, dk)
	if err := dkMapper.UnmapFromDynaKube(namespaces); err != nil {
		log.Error(err, "could not unmap dynakube from namespace")
	}
}

func (r *Reconciler) setupOneAgentInjection(ctx context.Context, dk *dynakube.DynaKube, dtClient *dynatrace.Client) error {
	log := logd.FromContext(ctx)

	err := r.versionReconciler.ReconcileCodeModules(ctx, dk, dtClient.Images, dtClient.Version)
	if err != nil {
		return err
	}

	err = r.connectionInfoReconciler.Reconcile(ctx, dtClient.OneAgent, dk)
	if err != nil {
		return err
	}

	err = r.istioReconciler.ReconcileCodeModules(ctx, dk)
	if err != nil {
		log.Error(err, "error reconciling istio configuration for codemodules")
	}

	if !dk.OneAgent().IsAppInjectionNeeded() {
		return nil
	}

	if dk.OneAgent().IsApplicationMonitoringMode() {
		dk.Status.SetPhase(status.Running)
	}

	setCodeModulesInjectionCreatedCondition(dk.Conditions())

	return nil
}

func (r *Reconciler) generateInitSecret(ctx context.Context, dtClient *dynatrace.Client, namespaces []corev1.Namespace, dk *dynakube.DynaKube) error {
	// The OneAgent mounts the bootstrapper secret when hostMonitoring is enabled.
	// The namespace selector is only available for appmon/CNFS, so hostmon without metadata enrichment will always have 0 namespaces here.
	if len(namespaces) > 0 || dk.OneAgent().IsHostMonitoringMode() {
		err := newBootstrapSecretGenerator(r.client, r.apiReader, dtClient.OneAgent).GenerateForDynakube(ctx, dk, namespaces)
		if err != nil {
			if k8sconditions.IsKubeAPIError(err) {
				k8sconditions.SetKubeAPIError(dk.Conditions(), codeModulesInjectionConditionType, err)
			}

			return err
		}
	}

	return nil
}

func (r *Reconciler) generateOTLPSecret(ctx context.Context, namespaces []corev1.Namespace, dk *dynakube.DynaKube) error {
	if len(namespaces) > 0 {
		err := newExporterSecretGenerator(r.client, r.apiReader).GenerateForDynakube(ctx, dk, namespaces)
		if err != nil {
			if k8sconditions.IsKubeAPIError(err) {
				k8sconditions.SetKubeAPIError(dk.Conditions(), otlpExporterConfigurationConditionType, err)
			}

			return err
		}
	}

	return nil
}

func (r *Reconciler) setupEnrichmentInjection(ctx context.Context, dk *dynakube.DynaKube, settingsClient dtsettings.Client) error {
	log := logd.FromContext(ctx)

	err := r.enrichmentRulesReconciler.Reconcile(ctx, settingsClient, dk)
	if err != nil {
		log.Info("couldn't reconcile metadata-enrichment rules")

		return err
	}

	if !dk.MetadataEnrichment().IsEnabled() {
		return nil
	}

	setMetadataEnrichmentCreatedCondition(dk.Conditions())

	return nil
}

func (r *Reconciler) createDynakubeMapper(ctx context.Context, dk *dynakube.DynaKube) *mapper.DynakubeMapper {
	operatorNamespace := dk.GetNamespace()
	dkMapper := mapper.NewDynakubeMapper(ctx, r.client, r.apiReader, operatorNamespace, dk)

	return &dkMapper
}

func (r *Reconciler) cleanupInitSecret(ctx context.Context, namespaces []corev1.Namespace, dk *dynakube.DynaKube) {
	log := logd.FromContext(ctx)

	if meta.FindStatusCondition(*dk.Conditions(), codeModulesInjectionConditionType) == nil &&
		meta.FindStatusCondition(*dk.Conditions(), metaDataEnrichmentConditionType) == nil &&
		meta.FindStatusCondition(*dk.Conditions(), bootstrapperconfig.ConfigConditionType) == nil {
		return
	}

	err := bootstrapperconfig.Cleanup(ctx, r.client, r.apiReader, namespaces, dk)
	if err != nil {
		log.Error(err, "failed to clean-up bootstrapper code module injection init-secrets")
	}

	meta.RemoveStatusCondition(dk.Conditions(), codeModulesInjectionConditionType)
	meta.RemoveStatusCondition(dk.Conditions(), metaDataEnrichmentConditionType)
	meta.RemoveStatusCondition(dk.Conditions(), bootstrapperconfig.ConfigConditionType)
}

func (r *Reconciler) cleanupOTLPSecret(ctx context.Context, namespaces []corev1.Namespace, dk *dynakube.DynaKube) {
	log := logd.FromContext(ctx)

	if meta.FindStatusCondition(*dk.Conditions(), otlpExporterConfigurationConditionType) == nil {
		return
	}

	err := exporterconfig.Cleanup(ctx, r.client, r.apiReader, namespaces, dk)
	if err != nil {
		log.Error(err, "failed to clean-up otlp exporter configuration secrets")
	}

	meta.RemoveStatusCondition(dk.Conditions(), otlpExporterConfigurationConditionType)
}

// Returns a copy of namespaces that only contains items that match the provided label selectors. The label selectors are ORed.
// If no label selectors were provided or any label selector matches all values (empty), the original list is returned unmodified.
// If an invalid label selector is encountered returns an empty list.
func filterNamespaces(ctx context.Context, namespaces []corev1.Namespace, labelSelectors ...*metav1.LabelSelector) []corev1.Namespace {
	selectors := make([]labels.Selector, 0, len(labelSelectors))

	var matchEverything bool

	for _, labelSelector := range labelSelectors {
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			// This should've been caught by validation. The operator might be in an invalid state. Do not replicate anything.
			// We do not need to report this error here, since the DynaKube mapper will already expose it to the user.
			logd.FromContext(ctx).Info("skipping secret replication for due to invalid selector", "selector", labelSelector)

			return nil
		}

		if selector.Empty() {
			matchEverything = true

			continue
		}

		if labels.MatchesNothing(selector) {
			continue
		}

		selectors = append(selectors, selector)
	}

	if matchEverything {
		return namespaces
	}

	var result []corev1.Namespace

	if len(selectors) > 0 {
		for _, ns := range namespaces {
			for _, selector := range selectors {
				if selector.Matches(labels.Set(ns.Labels)) {
					result = append(result, ns)

					break // continue with the next namespace
				}
			}
		}
	}

	return result
}
