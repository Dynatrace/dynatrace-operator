package dynakube

import (
	"context"
	goerrors "errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
)

func (controller *Controller) reconcileAppInjection(ctx context.Context, dynakube *dynakube.DynaKube, istioReconciler *istio.Reconciler, versionReconciler *version.Reconciler) error {
	if !dynakube.NeedAppInjection() {
		return controller.removeAppInjection(ctx, dynakube)
	}

	dkMapper := controller.createDynakubeMapper(ctx, dynakube)
	if err := dkMapper.MapFromDynakube(); err != nil {
		log.Info("update of a map of namespaces failed")
		return err
	}

	var setupErrors []error
	if err := controller.setupOneAgentInjection(ctx, dynakube, istioReconciler, versionReconciler); err != nil {
		setupErrors = append(setupErrors, err)
	}

	if err := controller.setupEnrichmentInjection(ctx, dynakube); err != nil {
		setupErrors = append(setupErrors, err)
	}
	if len(setupErrors) > 0 {
		return goerrors.Join(setupErrors...)
	}

	log.Info("app injection reconciled")
	return nil
}

func (controller *Controller) removeAppInjection(ctx context.Context, dynakube *dynakube.DynaKube) (err error) {
	dkMapper := controller.createDynakubeMapper(ctx, dynakube)

	if err := dkMapper.UnmapFromDynaKube(); err != nil {
		log.Info("could not unmap DynaKube from namespace")
		return err
	}

	endpointSecretGenerator := ingestendpoint.NewEndpointSecretGenerator(controller.client, controller.apiReader, dynakube.Namespace)
	err = endpointSecretGenerator.RemoveEndpointSecrets(ctx, dynakube)
	if err != nil {
		log.Info("could not remove data-ingest secret")
		return err
	}
	// TODO: remove initgeneration secret as well + handle errors jointly

	return nil
}

func (controller *Controller) setupOneAgentInjection(ctx context.Context, dynakube *dynakube.DynaKube, istioReconciler *istio.Reconciler, versionReconciler *version.Reconciler) error {
	if !dynakube.ApplicationMonitoringMode() && !dynakube.CloudNativeFullstackMode() {
		return nil
	}

	if istioReconciler != nil {
		err := istioReconciler.ReconcileCMCommunicationHosts(ctx, dynakube)
		if err != nil {
			return err
		}
	}
	err := versionReconciler.ReconcileCodeModules(ctx)
	if err != nil {
		return err
	}

	err = initgeneration.NewInitGenerator(controller.client, controller.apiReader, dynakube.Namespace).GenerateForDynakube(ctx, dynakube)
	if err != nil {
		log.Info("failed to generate init secret")
		return err
	}
	if dynakube.ApplicationMonitoringMode() {
		dynakube.Status.SetPhase(status.Running)
	}
	return nil
}

func (controller *Controller) setupEnrichmentInjection(ctx context.Context, dynakube *dynakube.DynaKube) error {
	if dynakube.FeatureDisableMetadataEnrichment() {
		return nil
	}
	endpointSecretGenerator := ingestendpoint.NewEndpointSecretGenerator(controller.client, controller.apiReader, dynakube.Namespace)
	err := endpointSecretGenerator.GenerateForDynakube(ctx, dynakube)
	if err != nil {
		log.Info("failed to generate data-ingest secret")
		return err
	}
	return nil
}

func (controller *Controller) createDynakubeMapper(ctx context.Context, dynakube *dynakube.DynaKube) *mapper.DynakubeMapper {
	dkMapper := mapper.NewDynakubeMapper(ctx, controller.client, controller.apiReader, controller.operatorNamespace, dynakube)
	return &dkMapper
}
