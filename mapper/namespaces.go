package mapper

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/go-logr/logr"
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
	logger     logr.Logger
}

func NewNamespaceMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, targetNs *corev1.Namespace, logger logr.Logger) NamespaceMapper {
	return NamespaceMapper{ctx, clt, apiReader, operatorNs, targetNs, logger}
}

// MapFromNamespace adds the labels to the targetNs if there is a matching Dynakube
func (nm NamespaceMapper) MapFromNamespace() (bool, error) {
	if nm.operatorNs == nm.targetNs.Name || isIgnoredNamespace(nm.targetNs.Name) {
		return false, nil
	}
	updatedNamespace, err := nm.updateNamespace()
	if err != nil {
		return false, err
	}
	return updatedNamespace, nil
}

func (nm NamespaceMapper) updateNamespace() (bool, error) {
	dkList := &dynatracev1beta1.DynaKubeList{}
	err := nm.client.List(nm.ctx, dkList)

	if err != nil {
		return false, errors.Cause(err)
	}

	return updateNamespace(nm.targetNs, dkList, nm.logger)
}
