package injection

import (
	"context"
	goerrors "errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/metadata/rules"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/otlp/exporterconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type istioReconciler interface {
	ReconcileCodeModules(ctx context.Context, dk *dynakube.DynaKube) error
}

type Reconciler struct {
	client                    client.Client
	apiReader                 client.Reader
	istioReconciler           istioReconciler
	versionReconciler         version.Reconciler
	connectionInfoReconciler  controllers.Reconciler
	enrichmentRulesReconciler controllers.Reconciler
}

func NewReconciler(
	client client.Client,
	apiReader client.Reader,
) *Reconciler {
	return &Reconciler{
		client:          client,
		apiReader:       apiReader,
		istioReconciler: istio.NewReconciler(client, apiReader),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dtClient *dynatrace.Client, dk *dynakube.DynaKube) error {
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
	versionReconciler := r.versionReconciler
	if versionReconciler == nil {
		versionReconciler = version.NewReconciler(r.apiReader, dtClient.Version, timeprovider.New().Freeze())
	}

	connectionInfoReconciler := r.connectionInfoReconciler
	if connectionInfoReconciler == nil {
		connectionInfoReconciler = oaconnectioninfo.NewReconciler(r.client, r.apiReader, dtClient.OneAgent, dk)
	}

	enrichmentRulesReconciler := r.enrichmentRulesReconciler
	if enrichmentRulesReconciler == nil {
		enrichmentRulesReconciler = rules.NewReconciler(dtClient.Settings, dk)
	}

	var setupErrors []error
	if err := r.setupOneAgentInjection(ctx, dk, versionReconciler, connectionInfoReconciler); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if err := r.setupEnrichmentInjection(ctx, dk, enrichmentRulesReconciler); err != nil {
		setupErrors = append(setupErrors, err)
	}

	return goerrors.Join(setupErrors...)
}

func (r *Reconciler) setupOTLPSecret(ctx context.Context, namespaces []corev1.Namespace, dk *dynakube.DynaKube) error {
	if dk.OTLPExporterConfiguration().IsEnabled() {
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
	if dk.OneAgent().IsAppInjectionNeeded() || dk.MetadataEnrichment().IsEnabled() {
		if err := r.generateInitSecret(ctx, dtClient, namespaces, dk); err != nil {
			return err
		}
	} else {
		r.cleanupInitSecret(ctx, namespaces, dk)
	}

	return nil
}

func (r *Reconciler) unmap(ctx context.Context, dk *dynakube.DynaKube) {
	namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, dk.Name)
	if err != nil {
		log.Error(err, "failed to list namespaces for dynakube", "dkName", dk.Name)
	}

	dkMapper := r.createDynakubeMapper(ctx, dk)
	if err := dkMapper.UnmapFromDynaKube(namespaces); err != nil {
		log.Error(err, "could not unmap dynakube from namespace", "dkName", dk.Name)
	}
}

func (r *Reconciler) setupOneAgentInjection(ctx context.Context, dk *dynakube.DynaKube, versionReconciler version.Reconciler, connectionInfoReconciler controllers.Reconciler) error {
	err := versionReconciler.ReconcileCodeModules(ctx, dk)
	if err != nil {
		return err
	}

	err = connectionInfoReconciler.Reconcile(ctx)
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
	err := bootstrapperconfig.NewSecretGenerator(r.client, r.apiReader, dtClient.OneAgent).GenerateForDynakube(ctx, dk, namespaces)
	if err != nil {
		if k8sconditions.IsKubeAPIError(err) {
			k8sconditions.SetKubeAPIError(dk.Conditions(), codeModulesInjectionConditionType, err)
		}

		return err
	}

	return nil
}

func (r *Reconciler) generateOTLPSecret(ctx context.Context, namespaces []corev1.Namespace, dk *dynakube.DynaKube) error {
	err := exporterconfig.NewSecretGenerator(r.client, r.apiReader).GenerateForDynakube(ctx, dk, namespaces)
	if err != nil {
		if k8sconditions.IsKubeAPIError(err) {
			k8sconditions.SetKubeAPIError(dk.Conditions(), otlpExporterConfigurationConditionType, err)
		}

		return err
	}

	return nil
}

func (r *Reconciler) setupEnrichmentInjection(ctx context.Context, dk *dynakube.DynaKube, enrichmentRulesReconciler controllers.Reconciler) error {
	err := enrichmentRulesReconciler.Reconcile(ctx)
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
	if meta.FindStatusCondition(*dk.Conditions(), codeModulesInjectionConditionType) == nil &&
		meta.FindStatusCondition(*dk.Conditions(), metaDataEnrichmentConditionType) == nil {
		return
	}

	err := bootstrapperconfig.Cleanup(ctx, r.client, r.apiReader, namespaces, dk)
	if err != nil {
		log.Error(err, "failed to clean-up bootstrapper code module injection init-secrets")
	}

	meta.RemoveStatusCondition(dk.Conditions(), codeModulesInjectionConditionType)
	meta.RemoveStatusCondition(dk.Conditions(), metaDataEnrichmentConditionType)
}

func (r *Reconciler) cleanupOTLPSecret(ctx context.Context, namespaces []corev1.Namespace, dk *dynakube.DynaKube) {
	if meta.FindStatusCondition(*dk.Conditions(), otlpExporterConfigurationConditionType) == nil {
		return
	}

	err := exporterconfig.Cleanup(ctx, r.client, r.apiReader, namespaces, dk)
	if err != nil {
		log.Error(err, "failed to clean-up otlp exporter configuration secrets")
	}

	meta.RemoveStatusCondition(dk.Conditions(), otlpExporterConfigurationConditionType)
}
