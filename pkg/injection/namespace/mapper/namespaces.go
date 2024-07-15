package mapper

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	injectionotel "github.com/Dynatrace/dynatrace-operator/pkg/injection/internal/otel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel/controller_runtime"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespaceMapper manages the mapping creation from the namespace's side
type NamespaceMapper struct {
	client     client.Client
	apiReader  client.Reader
	targetNs   *corev1.Namespace
	operatorNs string
}

func NewNamespaceMapper(clt client.Client, apiReader client.Reader, operatorNs string, targetNs *corev1.Namespace) NamespaceMapper {
	return NamespaceMapper{client: controller_runtime.NewClient(clt), apiReader: apiReader, operatorNs: operatorNs, targetNs: targetNs}
}

// MapFromNamespace adds the labels to the targetNs if there is a matching Dynakube
func (nm NamespaceMapper) MapFromNamespace(ctx context.Context) (bool, error) {
	ctx, span := dtotel.StartSpan(ctx, injectionotel.Tracer())
	defer span.End()

	updatedNamespace, err := nm.updateNamespace(ctx)
	if err != nil {
		span.RecordError(err)

		return false, err
	}

	return updatedNamespace, nil
}

func (nm NamespaceMapper) updateNamespace(ctx context.Context) (bool, error) {
	ctx, span := dtotel.StartSpan(ctx, injectionotel.Tracer())
	defer span.End()

	deployedDynakubes := &dynakube.DynaKubeList{}

	err := nm.client.List(ctx, deployedDynakubes)
	if err != nil {
		span.RecordError(err)

		return false, errors.Cause(err)
	}

	return updateNamespace(nm.targetNs, deployedDynakubes)
}
