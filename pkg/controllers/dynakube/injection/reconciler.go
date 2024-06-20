package injection

import (
	"context"
	goerrors "errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client                   client.Client
	apiReader                client.Reader
	dynakube                 *dynatracev1beta2.DynaKube
	istioReconciler          istio.Reconciler
	versionReconciler        version.Reconciler
	pmcSecretreconciler      controllers.Reconciler
	connectionInfoReconciler controllers.Reconciler
	dynatraceClient          dynatrace.Client
}

type ReconcilerBuilder func(
	client client.Client,
	apiReader client.Reader,
	dynatraceClient dynatrace.Client,
	istioClient *istio.Client,
	dynakube *dynatracev1beta2.DynaKube,
) controllers.Reconciler

//nolint:revive
func NewReconciler(
	client client.Client,
	apiReader client.Reader,
	dynatraceClient dynatrace.Client,
	istioClient *istio.Client,
	dynakube *dynatracev1beta2.DynaKube,
) controllers.Reconciler {
	var istioReconciler istio.Reconciler = nil

	if istioClient != nil {
		istioReconciler = istio.NewReconciler(istioClient)
	}

	return &reconciler{
		client:            client,
		apiReader:         apiReader,
		dynakube:          dynakube,
		istioReconciler:   istioReconciler,
		versionReconciler: version.NewReconciler(apiReader, dynatraceClient, timeprovider.New().Freeze()),
		pmcSecretreconciler: processmoduleconfigsecret.NewReconciler(
			client, apiReader, dynatraceClient, dynakube, timeprovider.New().Freeze()),
		connectionInfoReconciler: oaconnectioninfo.NewReconciler(client, apiReader, dynatraceClient, dynakube),
		dynatraceClient:          dynatraceClient,
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	err := r.versionReconciler.ReconcileCodeModules(ctx, r.dynakube)
	if err != nil {
		return err
	}

	err = r.reconcileIstioForCSIDriver(ctx)
	if err != nil {
		return err
	}

	err = r.connectionInfoReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	// do istio reconciliation for CodeModules here to enable cleanup of conditions
	if r.istioReconciler != nil {
		err = r.istioReconciler.ReconcileCodeModuleCommunicationHosts(ctx, r.dynakube)

		if err != nil {
			log.Error(err, "error reconciling istio configuration for codemodules")
		}
	}

	if !r.dynakube.NeedAppInjection() {
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

func (r *reconciler) reconcileIstioForCSIDriver(ctx context.Context) error {
	if r.istioReconciler != nil {
		err := r.istioReconciler.ReconcileCSIDriver(ctx, r.dynakube)
		if err != nil {
			return errors.WithMessage(err, "failed to reconcile istio objects for CSI Driver")
		}
	}

	return nil
}

func (r *reconciler) removeAppInjection(ctx context.Context) (err error) {
	namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, r.dynakube.Name)
	if err != nil {
		return errors.WithMessagef(err, "failed to list namespaces for dynakube %s", r.dynakube.Name)
	}

	dkMapper := r.createDynakubeMapper(ctx)

	if err := dkMapper.UnmapFromDynaKube(namespaces); err != nil {
		log.Info("could not unmap DynaKube from namespace")

		return err
	}

	endpointSecretGenerator := ingestendpoint.NewSecretGenerator(r.client, r.apiReader, r.dynakube.Namespace)

	err = endpointSecretGenerator.RemoveEndpointSecrets(ctx, namespaces)
	if err != nil {
		log.Info("could not remove metadata-enrichment secret")

		return err
	}
	// TODO: remove initgeneration secret as well + handle errors jointly

	return nil
}

func (r *reconciler) setupOneAgentInjection(ctx context.Context) error {
	if !r.dynakube.NeedAppInjection() {
		return nil
	}

	err := initgeneration.NewInitGenerator(r.client, r.apiReader, r.dynakube.Namespace).GenerateForDynakube(ctx, r.dynakube)
	if err != nil {
		log.Info("failed to generate init secret")

		return err
	}

	if r.dynakube.ApplicationMonitoringMode() {
		r.dynakube.Status.SetPhase(status.Running)
	}

	return nil
}

func (r *reconciler) setupEnrichmentInjection(ctx context.Context) error {
	if !r.dynakube.MetadataEnrichmentEnabled() {
		return nil
	}

	endpointSecretGenerator := ingestendpoint.NewSecretGenerator(r.client, r.apiReader, r.dynakube.Namespace)

	err := endpointSecretGenerator.GenerateForDynakube(ctx, r.dynakube)
	if err != nil {
		log.Info("failed to generate the metadata-enrichment secret")

		return err
	}

	return nil
}

func (r *reconciler) createDynakubeMapper(ctx context.Context) *mapper.DynakubeMapper {
	operatorNamespace := r.dynakube.GetNamespace()
	dkMapper := mapper.NewDynakubeMapper(ctx, r.client, r.apiReader, operatorNamespace, r.dynakube)

	return &dkMapper
}
