package mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	dynakube, err := nm.findDynakubesForNamespace()
	if err != nil {
		return err
	}

	if dynakube == nil {
		delete(nm.targetNs.Labels, InstanceLabel)
		return nil
	}
	nm.updateLabels(dynakube)
	return nil
}

// findDynakubesForNamespace tries to match the namespace to every dynakube with codeModules
// finds conflicting dynakubes(2 dynakube with codeModules on the same namespace)
func (nm NamespaceMapper) findDynakubesForNamespace() (*dynatracev1alpha1.DynaKube, error) {
	dynakubes := &dynatracev1alpha1.DynaKubeList{}
	err := nm.client.List(nm.ctx, dynakubes)

	if err != nil {
		return nil, errors.Cause(err)
	}

	var matchingDynakubes []dynatracev1alpha1.DynaKube

	conflict := ConflictChecker{}
	for _, dk := range dynakubes.Items {
		matches, err := nm.matchForDynakube(dk)
		if err != nil {
			return nil, err
		}
		if matches {
			if err := conflict.check(&dk); err != nil {
				return nil, err
			}
			matchingDynakubes = append(matchingDynakubes, dk)
		}
	}
	if len(matchingDynakubes) == 0 {
		nm.logger.Info("No matching dk found")
		return nil, nil
	}
	nm.logger.Info("Matching dk found", "dynakubes", matchingDynakubes[0].Name)
	return &matchingDynakubes[0], nil
}

func (nm NamespaceMapper) updateLabels(dynakube *dynatracev1alpha1.DynaKube) {
	if nm.targetNs.Labels == nil {
		nm.targetNs.Labels = make(map[string]string)
	}
	processedDks := map[string]bool{}
	oldDkName, ok := nm.targetNs.Labels[InstanceLabel]
	if !ok || oldDkName != dynakube.Name {
		processedDks[dynakube.Name] = true
		addNamespaceInjectLabel(dynakube.Name, nm.targetNs)
	}
}

func (nm NamespaceMapper) matchForDynakube(dk dynatracev1alpha1.DynaKube) (bool, error) {
	selector, err := metav1.LabelSelectorAsSelector(dk.Spec.MonitoredNamespaces)
	if err != nil {
		return false, errors.WithStack(err)
	}
	matches := selector.Matches(labels.Set(nm.targetNs.Labels))
	if matches && dk.Spec.CodeModules.Enabled {
		return matches, err
	}
	return false, nil
}
