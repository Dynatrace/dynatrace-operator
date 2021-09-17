package mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
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
func (nm NamespaceMapper) MapFromNamespace() error {
	if nm.operatorNs == nm.targetNs.Name {
		return nil
	}
	_, err := nm.findDynakubesForNamespace()
	if err != nil {
		return err
	}
	return nil
}

// findDynakubesForNamespace tries to match the namespace to every dynakube with codeModules
// finds conflicting dynakubes(2 dynakube with codeModules on the same namespace)
func (nm NamespaceMapper) findDynakubesForNamespace() (bool, error) {
	dkList := &dynatracev1alpha1.DynaKubeList{}
	err := nm.client.List(nm.ctx, dkList)

	if err != nil {
		return false, errors.Cause(err)
	}

	return checkDynakubes(nm.targetNs, dkList)
}
