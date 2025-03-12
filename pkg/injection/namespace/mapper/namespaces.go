package mapper

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
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
	return NamespaceMapper{client: clt, apiReader: apiReader, operatorNs: operatorNs, targetNs: targetNs}
}

// MapFromNamespace adds the labels to the targetNs if there is a matching Dynakube
func (nm NamespaceMapper) MapFromNamespace(ctx context.Context) (bool, error) {
	updatedNamespace, err := nm.updateNamespace(ctx)
	if err != nil {
		return false, err
	}

	return updatedNamespace, nil
}

func (nm NamespaceMapper) updateNamespace(ctx context.Context) (bool, error) {
	deployedDynakubes := &dynakube.DynaKubeList{}

	err := nm.client.List(ctx, deployedDynakubes)
	if err != nil {
		return false, errors.Cause(err)
	}

	return updateNamespace(nm.targetNs, deployedDynakubes)
}
