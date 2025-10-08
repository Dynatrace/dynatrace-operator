package otlp

import (
	"context"
	goerrors "errors"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/otlp/exporterconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client          client.Client
	apiReader       client.Reader
	dynatraceClient dynatrace.Client
	dk              *dynakube.DynaKube
}

type ReconcilerBuilder func(
	client client.Client,
	apiReader client.Reader,
	dynatraceClient dynatrace.Client,
	dk *dynakube.DynaKube,
) controllers.Reconciler

//nolint:revive
func NewReconciler(
	client client.Client,
	apiReader client.Reader,
	dynatraceClient dynatrace.Client,
	dk *dynakube.DynaKube,
) controllers.Reconciler {
	return &Reconciler{
		client:          client,
		apiReader:       apiReader,
		dynatraceClient: dynatraceClient,
		dk:              dk,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	var setupErrors []error

	// TODO use IsEnabled() function
	if r.dk.Spec.OTLPExporterConfiguration == nil {
		defer r.cleanup(ctx)
	} else {
		dkMapper := r.createDynakubeMapper(ctx)
		if err := dkMapper.MapFromDynakube(); err != nil {
			log.Info("update of a map of namespaces failed")

			setupErrors = append(setupErrors, err)
		}

		err := r.generateSecret(ctx)
		if err != nil {
			setupErrors = append(setupErrors, err)
		}
	}

	if len(setupErrors) > 0 {
		return goerrors.Join(setupErrors...)
	}

	setOTLPExporterConfigurationCondition(r.dk.Conditions())

	log.Info("OTLP exporter configuration reconciled")

	return nil
}

func (r *Reconciler) cleanup(ctx context.Context) {
	if meta.FindStatusCondition(*r.dk.Conditions(), otlpExporterConfigurationConditionType) == nil {
		return
	}
	defer meta.RemoveStatusCondition(r.dk.Conditions(), otlpExporterConfigurationConditionType)

	namespaces, err := mapper.GetNamespacesForDynakube(ctx, r.apiReader, r.dk.Name)
	if err != nil {
		log.Error(err, "failed to list namespaces for dynakube", "dkName", r.dk.Name)
	}

	err = exporterconfig.Cleanup(ctx, r.client, r.apiReader, namespaces, r.dk)
	if err != nil {
		log.Error(err, "failed to cleanup OTLP exporter configuration", "dkName", r.dk.Name)
	}
}

func (r *Reconciler) createDynakubeMapper(ctx context.Context) *mapper.DynakubeMapper {
	operatorNamespace := r.dk.GetNamespace()
	dkMapper := mapper.NewDynakubeMapper(ctx, r.client, r.apiReader, operatorNamespace, r.dk)

	return &dkMapper
}

func (r *Reconciler) generateSecret(ctx context.Context) error {
	err := exporterconfig.NewSecretGenerator(r.client, r.apiReader, r.dynatraceClient).GenerateForDynakube(ctx, r.dk)
	if err != nil {
		if conditions.IsKubeAPIError(err) {
			conditions.SetKubeAPIError(r.dk.Conditions(), otlpExporterConfigurationConditionType, err)
		}

		return err
	}

	return nil
}
