package injection

import (
	"context"
	goerrors "errors"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client              client.Client
	apiReader           client.Reader
	dynakube            *dynatracev1beta1.DynaKube
	istioReconciler     istio.Reconciler
	versionReconciler   version.Reconciler
	pmcSecretreconciler controllers.Reconciler
	connectionInfoReconciler controllers.Reconciler
	dynatraceClient     dynatrace.Client
	scheme              *runtime.Scheme
}

type ReconcilerBuilder func(
	client client.Client,
	apiReader client.Reader,
	scheme *runtime.Scheme,
	dynatraceClient dynatrace.Client,
	istioClient *istio.Client,
	fs afero.Afero,
	dynakube *dynatracev1beta1.DynaKube,
) controllers.Reconciler

//nolint:revive
func NewReconciler(
	client client.Client,
	apiReader client.Reader,
	scheme *runtime.Scheme,
	dynatraceClient dynatrace.Client,
	istioClient *istio.Client,
	fs afero.Afero,
	dynakube *dynatracev1beta1.DynaKube,
) controllers.Reconciler {
	var istioReconciler istio.Reconciler = nil

	if istioClient != nil {
		istioReconciler = istio.NewReconciler(istioClient)
	}

	return &reconciler{
		client:            client,
		apiReader:         apiReader,
		scheme:          scheme,
		dynakube:          dynakube,
		istioReconciler:   istioReconciler,
		versionReconciler: version.NewReconciler(apiReader, dynatraceClient, fs, timeprovider.New().Freeze()),
		pmcSecretreconciler: processmoduleconfigsecret.NewReconciler(
			client, apiReader, dynatraceClient, dynakube, scheme, timeprovider.New().Freeze()),
		connectionInfoReconciler: oaconnectioninfo.NewReconciler(client, apiReader, scheme, dynatraceClient, dynakube),
		dynatraceClient: dynatraceClient,
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	if !r.dynakube.NeedAppInjection() {
		return r.removeAppInjection(ctx)
	}

	err := r.connectionInfoReconciler.Reconcile(ctx)
	if err != nil {
		return err
	} // TODO: there tends to be a clean up for each reconcileX function, so it might makes sense to have the same here

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

	err := r.pmcSecretreconciler.Reconcile(ctx)
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
	dkMapper := r.createDynakubeMapper(ctx)

	if err := dkMapper.UnmapFromDynaKube(); err != nil {
		log.Info("could not unmap DynaKube from namespace")

		return err
	}

	endpointSecretGenerator := ingestendpoint.NewEndpointSecretGenerator(r.client, r.apiReader, r.dynakube.Namespace)

	err = endpointSecretGenerator.RemoveEndpointSecrets(ctx, r.dynakube)
	if err != nil {
		log.Info("could not remove data-ingest secret")

		return err
	}
	// TODO: remove initgeneration secret as well + handle errors jointly

	return nil
}

func (r *reconciler) setupOneAgentInjection(ctx context.Context) error {
	if !r.dynakube.ApplicationMonitoringMode() && !r.dynakube.CloudNativeFullstackMode() {
		return nil
	}

	if r.istioReconciler != nil {
		err := r.istioReconciler.ReconcileCodeModuleCommunicationHosts(ctx, r.dynakube)
		if err != nil {
			return err
		}
	}

	err := r.versionReconciler.ReconcileCodeModules(ctx, r.dynakube)
	if err != nil {
		return err
	}

	err = initgeneration.NewInitGenerator(r.client, r.apiReader, r.dynakube.Namespace).GenerateForDynakube(ctx, r.dynakube)
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
	if r.dynakube.FeatureDisableMetadataEnrichment() {
		return nil
	}

	endpointSecretGenerator := ingestendpoint.NewEndpointSecretGenerator(r.client, r.apiReader, r.dynakube.Namespace)

	err := endpointSecretGenerator.GenerateForDynakube(ctx, r.dynakube)
	if err != nil {
		log.Info("failed to generate data-ingest secret")

		return err
	}

	return nil
}

func (r *reconciler) createDynakubeMapper(ctx context.Context) *mapper.DynakubeMapper {
	operatorNamespace := r.dynakube.GetNamespace()
	dkMapper := mapper.NewDynakubeMapper(ctx, r.client, r.apiReader, operatorNamespace, r.dynakube)

	return &dkMapper
}
