package injection

import (
	"context"
	goerrors "errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/metadata/rules"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/monitoredentities"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client                      client.Client
	apiReader                   client.Reader
	dk                          *dynakube.DynaKube
	istioReconciler             istio.Reconciler
	versionReconciler           version.Reconciler
	pmcSecretreconciler         controllers.Reconciler
	connectionInfoReconciler    controllers.Reconciler
	monitoredEntitiesReconciler controllers.Reconciler
	enrichmentRulesReconciler   controllers.Reconciler
	dynatraceClient             dynatrace.Client
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

	return &reconciler{
		client:            client,
		apiReader:         apiReader,
		dk:                dk,
		dynatraceClient:   dynatraceClient,
		istioReconciler:   istioReconciler,
		versionReconciler: version.NewReconciler(apiReader, dynatraceClient, timeprovider.New().Freeze()),
		pmcSecretreconciler: processmoduleconfigsecret.NewReconciler(
			client, apiReader, dynatraceClient, dk, timeprovider.New().Freeze()),
		connectionInfoReconciler:    oaconnectioninfo.NewReconciler(client, apiReader, dynatraceClient, dk),
		enrichmentRulesReconciler:   rules.NewReconciler(dynatraceClient, dk),
		monitoredEntitiesReconciler: monitoredentities.NewReconciler(dynatraceClient, dk),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	// because the 2 injection type we have share the label that the webhook is listening to, we can only clean that label up if both are disabled
	// but we should only clean-up the labels after everything else is cleaned up because the clean-up for the secrets depend on the label still being there
	// but we have to do the mapping before everything when its necessary
	if !r.dk.OneAgent().IsAppInjectionNeeded() && !r.dk.MetadataEnrichmentEnabled() {
		defer r.unmapDynakube(ctx)
	} else {
		dkMapper := r.createDynakubeMapper(ctx)
		if err := dkMapper.MapFromDynakube(); err != nil {
			log.Info("update of a map of namespaces failed")

			return err
		}
	}

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

	log.Info("app injection reconciled")

	return nil
}

func (r *reconciler) unmapDynakube(ctx context.Context) {
	if meta.FindStatusCondition(*r.dk.Conditions(), codeModulesInjectionConditionType) != nil &&
		meta.FindStatusCondition(*r.dk.Conditions(), metaDataEnrichmentConditionType) != nil {
		return
	}

	namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, r.dk.Name)
	if err != nil {
		log.Error(err, "failed to list namespaces for dynakube", "dkName", r.dk.Name)
	}

	dkMapper := r.createDynakubeMapper(ctx)
	if err := dkMapper.UnmapFromDynaKube(namespaces); err != nil {
		log.Error(err, "could not unmap dynakube from namespace", "dkName", r.dk.Name)
	}
}

func (r *reconciler) setupOneAgentInjection(ctx context.Context) error {
	err := r.versionReconciler.ReconcileCodeModules(ctx, r.dk)
	if err != nil {
		return err
	}

	err = r.connectionInfoReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	if !r.dk.FeatureDownloadViaJob() || r.dk.OneAgent().IsCSIAvailable() {
		err = r.pmcSecretreconciler.Reconcile(ctx)
		if err != nil {
			return err
		}
	}

	if r.istioReconciler != nil {
		err = r.istioReconciler.ReconcileCodeModuleCommunicationHosts(ctx, r.dk)

		if err != nil {
			log.Error(err, "error reconciling istio configuration for codemodules")
		}
	}

	if !r.dk.OneAgent().IsAppInjectionNeeded() {
		r.cleanupOneAgentInjection(ctx)

		return nil
	}

	if r.dk.FeatureDownloadViaJob() && !r.dk.OneAgent().IsCSIAvailable() {
		err = bootstrapperconfig.NewBootstrapperInitGenerator(r.client, r.apiReader, r.dynatraceClient, r.dk.Namespace).GenerateForDynakube(ctx, r.dk)
		if err != nil {
			if conditions.IsKubeApiError(err) {
				conditions.SetKubeApiError(r.dk.Conditions(), codeModulesInjectionConditionType, err)
			}

			return err
		}
	} else {
		err = initgeneration.NewInitGenerator(r.client, r.apiReader, r.dk.Namespace).GenerateForDynakube(ctx, r.dk)
		if err != nil {
			if conditions.IsKubeApiError(err) {
				conditions.SetKubeApiError(r.dk.Conditions(), codeModulesInjectionConditionType, err)
			}

			return err
		}
	}

	if r.dk.OneAgent().IsApplicationMonitoringMode() {
		r.dk.Status.SetPhase(status.Running)
	}

	setCodeModulesInjectionCreatedCondition(r.dk.Conditions())

	return nil
}

func (r *reconciler) cleanupOneAgentInjection(ctx context.Context) {
	if meta.FindStatusCondition(*r.dk.Conditions(), codeModulesInjectionConditionType) != nil {
		defer meta.RemoveStatusCondition(r.dk.Conditions(), codeModulesInjectionConditionType)

		namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, r.dk.Name)
		if err != nil {
			log.Error(err, "failed to list injected namespace during code module injection cleanup")

			return
		}

		if r.dk.FeatureDownloadViaJob() && !r.dk.OneAgent().IsCSIAvailable() {
			err = bootstrapperconfig.NewBootstrapperInitGenerator(r.client, r.apiReader, r.dynatraceClient, r.dk.Namespace).Cleanup(ctx, namespaces)
			if err != nil {
				log.Error(err, "failed to clean-up bootstrapper code module injection init-secrets")
			}
		} else {
			err = initgeneration.NewInitGenerator(r.client, r.apiReader, r.dk.Namespace).Cleanup(ctx, namespaces)
			if err != nil {
				log.Error(err, "failed to clean-up code module injection init-secrets")
			}
		}
	}
}

func (r *reconciler) setupEnrichmentInjection(ctx context.Context) error {
	err := r.monitoredEntitiesReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	err = r.enrichmentRulesReconciler.Reconcile(ctx)
	if err != nil {
		log.Info("couldn't reconcile metadata-enrichment rules")

		return err
	}

	if !r.dk.MetadataEnrichmentEnabled() {
		r.cleanupEnrichmentInjection(ctx)

		return nil
	}

	endpointSecretGenerator := ingestendpoint.NewSecretGenerator(r.client, r.apiReader, r.dk.Namespace)

	err = endpointSecretGenerator.GenerateForDynakube(ctx, r.dk)
	if err != nil {
		if conditions.IsKubeApiError(err) {
			conditions.SetKubeApiError(r.dk.Conditions(), metaDataEnrichmentConditionType, err)
		}

		return err
	}

	setMetadataEnrichmentCreatedCondition(r.dk.Conditions())

	return nil
}

func (r *reconciler) cleanupEnrichmentInjection(ctx context.Context) {
	if meta.FindStatusCondition(*r.dk.Conditions(), metaDataEnrichmentConditionType) != nil {
		defer meta.RemoveStatusCondition(r.dk.Conditions(), metaDataEnrichmentConditionType)

		namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, r.dk.Name)
		if err != nil {
			log.Error(err, "failed to list injected namespace during metadata-enrichment injection cleanup")

			return
		}

		err = ingestendpoint.NewSecretGenerator(r.client, r.apiReader, r.dk.Namespace).RemoveEndpointSecrets(ctx, namespaces)
		if err != nil {
			log.Error(err, "failed to clean-up metadata-enrichment injection secrets")
		}
	}
}

func (r *reconciler) createDynakubeMapper(ctx context.Context) *mapper.DynakubeMapper {
	operatorNamespace := r.dk.GetNamespace()
	dkMapper := mapper.NewDynakubeMapper(ctx, r.client, r.apiReader, operatorNamespace, r.dk)

	return &dkMapper
}
