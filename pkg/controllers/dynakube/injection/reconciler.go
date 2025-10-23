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
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/k8sentity"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/metadata/rules"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/otlp/exporterconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client                    client.Client
	apiReader                 client.Reader
	dk                        *dynakube.DynaKube
	istioReconciler           istio.Reconciler
	versionReconciler         version.Reconciler
	connectionInfoReconciler  controllers.Reconciler
	k8sEntityReconciler       controllers.Reconciler
	enrichmentRulesReconciler controllers.Reconciler
	dynatraceClient           dynatrace.Client
}

type ReconcilerBuilder func(
	client client.Client,
	apiReader client.Reader,
	dynatraceClient dynatrace.Client,
	istioClient *istio.Client,
	dk *dynakube.DynaKube,
) controllers.Reconciler

//nolint:revive
func NewReconciler(
	client client.Client,
	apiReader client.Reader,
	dynatraceClient dynatrace.Client,
	istioClient *istio.Client,
	dk *dynakube.DynaKube,
) controllers.Reconciler {
	var istioReconciler istio.Reconciler = nil

	if istioClient != nil {
		istioReconciler = istio.NewReconciler(istioClient)
	}

	return &Reconciler{
		client:                    client,
		apiReader:                 apiReader,
		dk:                        dk,
		dynatraceClient:           dynatraceClient,
		istioReconciler:           istioReconciler,
		versionReconciler:         version.NewReconciler(apiReader, dynatraceClient, timeprovider.New().Freeze()),
		connectionInfoReconciler:  oaconnectioninfo.NewReconciler(client, apiReader, dynatraceClient, dk),
		enrichmentRulesReconciler: rules.NewReconciler(dynatraceClient, dk),
		k8sEntityReconciler:       k8sentity.NewReconciler(dynatraceClient, dk),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	var setupErrors []error
	if err := r.setupOneAgentInjection(ctx); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if err := r.setupEnrichmentInjection(ctx); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if len(setupErrors) > 0 {
		return goerrors.Join(setupErrors...)
	}

	if !r.dk.OneAgent().IsAppInjectionNeeded() && !r.dk.MetadataEnrichment().IsEnabled() && !r.dk.OTLPExporterConfiguration().IsEnabled() {
		defer r.unmap(ctx)
	} else {
		dkMapper := r.createDynakubeMapper(ctx)

		if err := dkMapper.MapFromDynakube(); err != nil {
			log.Info("update of a map of namespaces failed")

			setupErrors = append(setupErrors, err)
		}

		err := r.generateInitSecret(ctx)
		if err != nil {
			setupErrors = append(setupErrors, err)
		}
	}

	namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, r.dk.Name)
	if err != nil {
		return err
	}

	if r.dk.OneAgent().IsAppInjectionNeeded() || r.dk.MetadataEnrichment().IsEnabled() {
		err := r.generateInitSecret(ctx)
		if err != nil {
			setupErrors = append(setupErrors, err)
		}
	} else {
		r.cleanupInitSecret(ctx, namespaces)
	}

	if r.dk.OTLPExporterConfiguration().IsEnabled() {
		err := r.generateOTLPSecret(ctx, namespaces)
		if err != nil {
			setupErrors = append(setupErrors, err)
		} else {
			setOTLPExporterConfigurationCondition(r.dk.Conditions())
		}
	} else {
		r.cleanupOTLPSecret(ctx, namespaces)
	}

	if len(setupErrors) > 0 {
		return goerrors.Join(setupErrors...)
	}

	log.Info("app injection reconciled")

	return nil
}

func (r *Reconciler) unmap(ctx context.Context) {
	namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, r.dk.Name)
	if err != nil {
		log.Error(err, "failed to list namespaces for dynakube", "dkName", r.dk.Name)
	}

	dkMapper := r.createDynakubeMapper(ctx)
	if err := dkMapper.UnmapFromDynaKube(namespaces); err != nil {
		log.Error(err, "could not unmap dynakube from namespace", "dkName", r.dk.Name)
	}

}

func (r *Reconciler) setupOneAgentInjection(ctx context.Context) error {
	err := r.versionReconciler.ReconcileCodeModules(ctx, r.dk)
	if err != nil {
		return err
	}

	err = r.connectionInfoReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	if r.istioReconciler != nil {
		err = r.istioReconciler.ReconcileCodeModuleCommunicationHosts(ctx, r.dk)
		if err != nil {
			log.Error(err, "error reconciling istio configuration for codemodules")
		}
	}

	if !r.dk.OneAgent().IsAppInjectionNeeded() {
		return nil
	}

	if r.dk.OneAgent().IsApplicationMonitoringMode() {
		r.dk.Status.SetPhase(status.Running)
	}

	setCodeModulesInjectionCreatedCondition(r.dk.Conditions())

	return nil
}

func (r *Reconciler) generateInitSecret(ctx context.Context) error {
	err := bootstrapperconfig.NewSecretGenerator(r.client, r.apiReader, r.dynatraceClient).GenerateForDynakube(ctx, r.dk)
	if err != nil {
		if conditions.IsKubeAPIError(err) {
			conditions.SetKubeAPIError(r.dk.Conditions(), codeModulesInjectionConditionType, err)
		}

		return err
	}

	return nil
}

func (r *Reconciler) generateOTLPSecret(ctx context.Context, namespaces []corev1.Namespace) error {
	err := exporterconfig.NewSecretGenerator(r.client, r.apiReader, r.dynatraceClient).GenerateForDynakube(ctx, r.dk, namespaces)
	if err != nil {
		if conditions.IsKubeAPIError(err) {
			conditions.SetKubeAPIError(r.dk.Conditions(), otlpExporterConfigurationConditionType, err)
		}

		return err
	}

	return nil
}

func (r *Reconciler) setupEnrichmentInjection(ctx context.Context) error {
	err := r.k8sEntityReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	err = r.enrichmentRulesReconciler.Reconcile(ctx)
	if err != nil {
		log.Info("couldn't reconcile metadata-enrichment rules")

		return err
	}

	if !r.dk.MetadataEnrichment().IsEnabled() {
		return nil
	}

	setMetadataEnrichmentCreatedCondition(r.dk.Conditions())

	return nil
}

func (r *Reconciler) createDynakubeMapper(ctx context.Context) *mapper.DynakubeMapper {
	operatorNamespace := r.dk.GetNamespace()
	dkMapper := mapper.NewDynakubeMapper(ctx, r.client, r.apiReader, operatorNamespace, r.dk)

	return &dkMapper
}

func (r *Reconciler) cleanupInitSecret(ctx context.Context, namespaces []corev1.Namespace) {
	if meta.FindStatusCondition(*r.dk.Conditions(), codeModulesInjectionConditionType) == nil &&
		meta.FindStatusCondition(*r.dk.Conditions(), metaDataEnrichmentConditionType) == nil {
		return
	}
	defer meta.RemoveStatusCondition(r.dk.Conditions(), codeModulesInjectionConditionType)
	defer meta.RemoveStatusCondition(r.dk.Conditions(), metaDataEnrichmentConditionType)

	err := bootstrapperconfig.Cleanup(ctx, r.client, r.apiReader, namespaces, r.dk)
	if err != nil {
		log.Error(err, "failed to clean-up bootstrapper code module injection init-secrets")
	}
}

func (r *Reconciler) cleanupOTLPSecret(ctx context.Context, namespaces []corev1.Namespace) {
	if meta.FindStatusCondition(*r.dk.Conditions(), otlpExporterConfigurationConditionType) == nil {
		return
	}

	err := exporterconfig.Cleanup(ctx, r.client, r.apiReader, namespaces, r.dk)
	if err != nil {
		log.Error(err, "failed to clean-up otlp exporter configuration secrets")
	}

	meta.RemoveStatusCondition(r.dk.Conditions(), otlpExporterConfigurationConditionType)
}
