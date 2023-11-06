package mapper

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	injectionotel "github.com/Dynatrace/dynatrace-operator/pkg/injection/otel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/otel/controller_runtime"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespaceMapper manages the mapping creation from the namespace's side
type NamespaceMapper struct {
	ctx        context.Context
	client     client.Client
	apiReader  client.Reader
	operatorNs string
	targetNs   *corev1.Namespace
}

func NewNamespaceMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, targetNs *corev1.Namespace) NamespaceMapper { //nolint:revive // argument-limit doesn't apply to constructors
	return NamespaceMapper{ctx, controller_runtime.NewClient(clt), apiReader, operatorNs, targetNs}
}

// MapFromNamespace adds the labels to the targetNs if there is a matching Dynakube
func (nm NamespaceMapper) MapFromNamespace(ctx context.Context) (bool, error) {
	ctx, span := injectionotel.StartSpan(ctx, "NamespaceMapper.MapFromNamespace")
	defer span.End()

	updatedNamespace, err := nm.updateNamespace(ctx)
	if err != nil {
		return false, err
	}
	return updatedNamespace, nil
}

func (nm NamespaceMapper) updateNamespace(ctx context.Context) (bool, error) {
	ctx, span := injectionotel.StartSpan(ctx, "NamespaceMapper.updateNamespace")
	defer span.End()

	deployedDynakubes := &dynatracev1beta1.DynaKubeList{}
	err := nm.client.List(ctx, deployedDynakubes)

	if err != nil {
		span.RecordError(err)
		return false, errors.Cause(err)
	}

	return updateNamespace(nm.targetNs, deployedDynakubes)
}
