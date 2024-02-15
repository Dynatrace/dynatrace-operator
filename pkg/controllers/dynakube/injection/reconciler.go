package injection

import (
	"context"
	goerrors "errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client            client.Client
	apiReader         client.Reader
	dynakube          *dynatracev1beta1.DynaKube
	istioReconciler   istio.Reconciler
	versionReconciler version.Reconciler
}

type ReconcilerBuilder func(
	client client.Client,
	apiReader client.Reader,
	dynatraceClient dynatrace.Client,
	istioClient *istio.Client,
	fs afero.Afero,
	dynakube *dynatracev1beta1.DynaKube,
) controllers.Reconciler

//nolint:revive
func NewReconciler(
	client client.Client,
	apiReader client.Reader,
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
		dynakube:          dynakube,
		istioReconciler:   istioReconciler,
		versionReconciler: version.NewReconciler(apiReader, dynatraceClient, fs, timeprovider.New().Freeze()),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	if r.istioReconciler != nil {
		err := r.istioReconciler.ReconcileAPIUrl(ctx, r.dynakube)
		if err != nil {
			return errors.WithMessage(err, "failed to reconcile istio objects for API url")
		}
	}

	if !r.dynakube.NeedAppInjection() {
		return r.removeAppInjection(ctx, r.dynakube)
	}

	dkMapper := r.createDynakubeMapper(ctx, r.dynakube)
	if err := dkMapper.MapFromDynakube(); err != nil {
		log.Info("update of a map of namespaces failed")
		return err
	}

	var setupErrors []error
	if err := r.setupOneAgentInjection(ctx, r.dynakube, r.istioReconciler, r.versionReconciler); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if err := r.setupEnrichmentInjection(ctx, r.dynakube); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if len(setupErrors) > 0 {
		return goerrors.Join(setupErrors...)
	}

	log.Info("app injection reconciled")

	return nil
}

func (r *reconciler) removeAppInjection(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (err error) {
	dkMapper := r.createDynakubeMapper(ctx, dynakube)

	if err := dkMapper.UnmapFromDynaKube(); err != nil {
		log.Info("could not unmap DynaKube from namespace")
		return err
	}

	endpointSecretGenerator := ingestendpoint.NewEndpointSecretGenerator(r.client, r.apiReader, dynakube.Namespace)

	err = endpointSecretGenerator.RemoveEndpointSecrets(ctx, dynakube)
	if err != nil {
		log.Info("could not remove data-ingest secret")
		return err
	}
	// TODO: remove initgeneration secret as well + handle errors jointly

	return nil
}

func (r *reconciler) setupOneAgentInjection(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, istioReconciler istio.Reconciler, versionReconciler version.Reconciler) error {
	if !dynakube.ApplicationMonitoringMode() && !dynakube.CloudNativeFullstackMode() {
		return nil
	}

	if istioReconciler != nil {
		err := istioReconciler.ReconcileCodeModuleCommunicationHosts(ctx, dynakube)
		if err != nil {
			return err
		}
	}

	err := versionReconciler.ReconcileCodeModules(ctx, dynakube)
	if err != nil {
		return err
	}

	err = initgeneration.NewInitGenerator(r.client, r.apiReader, dynakube.Namespace).GenerateForDynakube(ctx, dynakube)
	if err != nil {
		log.Info("failed to generate init secret")
		return err
	}

	if dynakube.ApplicationMonitoringMode() {
		dynakube.Status.SetPhase(status.Running)
	}

	return nil
}

func (r *reconciler) setupEnrichmentInjection(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	if dynakube.FeatureDisableMetadataEnrichment() {
		return nil
	}

	endpointSecretGenerator := ingestendpoint.NewEndpointSecretGenerator(r.client, r.apiReader, dynakube.Namespace)

	err := endpointSecretGenerator.GenerateForDynakube(ctx, dynakube)
	if err != nil {
		log.Info("failed to generate data-ingest secret")
		return err
	}

	return nil
}

func (r *reconciler) createDynakubeMapper(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) *mapper.DynakubeMapper {
	operatorNamespace := r.dynakube.GetNamespace()
	dkMapper := mapper.NewDynakubeMapper(ctx, r.client, r.apiReader, operatorNamespace, dynakube)

	return &dkMapper
}
