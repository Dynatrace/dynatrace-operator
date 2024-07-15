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
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client                    client.Client
	apiReader                 client.Reader
	dk                        *dynakube.DynaKube
	istioReconciler           istio.Reconciler
	versionReconciler         version.Reconciler
	pmcSecretreconciler       controllers.Reconciler
	connectionInfoReconciler  controllers.Reconciler
	enrichmentRulesReconciler controllers.Reconciler
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
		istioReconciler:   istioReconciler,
		versionReconciler: version.NewReconciler(apiReader, dynatraceClient, timeprovider.New().Freeze()),
		pmcSecretreconciler: processmoduleconfigsecret.NewReconciler(
			client, apiReader, dynatraceClient, dk, timeprovider.New().Freeze()),
		connectionInfoReconciler:  oaconnectioninfo.NewReconciler(client, apiReader, dynatraceClient, dk),
		enrichmentRulesReconciler: rules.NewReconciler(dynatraceClient, dk),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	err := r.versionReconciler.ReconcileCodeModules(ctx, r.dk)
	if err != nil {
		return err
	}

	err = r.connectionInfoReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	// do istio reconciliation for CodeModules here to enable cleanup of conditions
	if r.istioReconciler != nil {
		err = r.istioReconciler.ReconcileCodeModuleCommunicationHosts(ctx, r.dk)

		if err != nil {
			log.Error(err, "error reconciling istio configuration for codemodules")
		}
	}

	err = r.enrichmentRulesReconciler.Reconcile(ctx)
	if err != nil {
		log.Info("couldn't reconcile metadata-enrichment rules")

		return err
	}

	if !r.dk.NeedAppInjection() {
		return r.removeAppInjection(ctx)
	}

	dkMapper := r.createDynakubeMapper(ctx)
	if err := dkMapper.MapFromDynakube(); err != nil {
		log.Info("update of a map of namespaces failed")

		return err
	}

	var setupErrors []error
	if err := r.setupOneAgentInjection(ctx); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if err := r.setupEnrichmentInjection(ctx); err != nil {
		setupErrors = append(setupErrors, err)
	}

	err = r.pmcSecretreconciler.Reconcile(ctx)
	if err != nil {
		setupErrors = append(setupErrors, err)
	}

	if len(setupErrors) > 0 {
		return goerrors.Join(setupErrors...)
	}

	log.Info("app injection reconciled")

	return nil
}

func (r *reconciler) removeAppInjection(ctx context.Context) (err error) {
	namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, r.dk.Name)
	if err != nil {
		return errors.WithMessagef(err, "failed to list namespaces for dynakube %s", r.dk.Name)
	}

	dkMapper := r.createDynakubeMapper(ctx)

	if err := dkMapper.UnmapFromDynaKube(namespaces); err != nil {
		log.Info("could not unmap DynaKube from namespace")

		return err
	}

	r.cleanupEnrichmentInjection(ctx, namespaces)
	r.cleanupOneAgentInjection(ctx, namespaces)

	return nil
}

func (r *reconciler) setupOneAgentInjection(ctx context.Context) error {
	if !r.dk.NeedAppInjection() {
		return nil
	}

	err := initgeneration.NewInitGenerator(r.client, r.apiReader, r.dk.Namespace).GenerateForDynakube(ctx, r.dk)
	if err != nil {
		if conditions.IsKubeApiError(err) {
			conditions.SetKubeApiError(r.dk.Conditions(), codeModulesInjectionConditionType, err)
		}

		return err
	}

	if r.dk.ApplicationMonitoringMode() {
		r.dk.Status.SetPhase(status.Running)
	}

	setCodeModulesInjectionCreatedCondition(r.dk.Conditions())

	return nil
}

func (r *reconciler) cleanupOneAgentInjection(ctx context.Context, namespaces []corev1.Namespace) {
	errs := make([]error, 0)

	if meta.FindStatusCondition(*r.dk.Conditions(), codeModulesInjectionConditionType) != nil {
		err := initgeneration.NewInitGenerator(r.client, r.apiReader, r.dk.Namespace).Cleanup(ctx, namespaces)
		if err != nil {
			errs = append(errs, err)
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), codeModulesInjectionConditionType)
	}

	log.Error(goerrors.Join(errs...), "failed to clean-up code module injection")
}

func (r *reconciler) setupEnrichmentInjection(ctx context.Context) error {
	if !r.dk.MetadataEnrichmentEnabled() {
		return nil
	}

	endpointSecretGenerator := ingestendpoint.NewSecretGenerator(r.client, r.apiReader, r.dk.Namespace)

	err := endpointSecretGenerator.GenerateForDynakube(ctx, r.dk)
	if err != nil {
		if conditions.IsKubeApiError(err) {
			conditions.SetKubeApiError(r.dk.Conditions(), metaDataEnrichmentConditionType, err)
		}

		return err
	}

	setMetadataEnrichmentCreatedCondition(r.dk.Conditions())

	return nil
}

func (r *reconciler) cleanupEnrichmentInjection(ctx context.Context, namespaces []corev1.Namespace) {
	if meta.FindStatusCondition(*r.dk.Conditions(), metaDataEnrichmentConditionType) != nil {
		err := ingestendpoint.NewSecretGenerator(r.client, r.apiReader, r.dk.Namespace).RemoveEndpointSecrets(ctx, namespaces)
		if err != nil {
			log.Error(err, "failed to clean-up metadata-enrichment secrets")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), metaDataEnrichmentConditionType)
	}
}

func (r *reconciler) createDynakubeMapper(ctx context.Context) *mapper.DynakubeMapper {
	operatorNamespace := r.dk.GetNamespace()
	dkMapper := mapper.NewDynakubeMapper(ctx, r.client, r.apiReader, operatorNamespace, r.dk)

	return &dkMapper
}
