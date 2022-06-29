package mapper

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
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

func NewNamespaceMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, targetNs *corev1.Namespace) NamespaceMapper {
	return NamespaceMapper{ctx, clt, apiReader, operatorNs, targetNs}
}

// MapFromNamespace adds the labels to the targetNs if there is a matching Dynakube
func (nm NamespaceMapper) MapFromNamespace() (bool, error) {
	updatedNamespace, err := nm.updateNamespace()
	if err != nil {
		return false, err
	}
	return updatedNamespace, nil
}

func (nm NamespaceMapper) updateNamespace() (bool, error) {
	deployedDynakubes := &dynatracev1beta1.DynaKubeList{}
	err := nm.client.List(nm.ctx, deployedDynakubes)

	if err != nil {
		return false, errors.Cause(err)
	}

	return updateNamespace(nm.targetNs, deployedDynakubes)
}
