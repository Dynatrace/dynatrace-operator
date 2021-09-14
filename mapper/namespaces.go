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

func (nm NamespaceMapper) MapFromNamespace() error {
	if nm.operatorNs == nm.targetNs.Name {
		return nil
	}
	dynakubes, err := nm.findDynakubesForNamespace()
	if err != nil {
		return err
	}

	if dynakubes == nil {
		delete(nm.targetNs.Labels, InstanceLabel)
		return nil
	}
	nm.updateLabels(dynakubes)
	return nil
}

func (nm NamespaceMapper) findDynakubesForNamespace() ([]dynatracev1alpha1.DynaKube, error) {
	dynakubes := &dynatracev1alpha1.DynaKubeList{}
	err := nm.client.List(nm.ctx, dynakubes)

	if err != nil {
		return nil, errors.Cause(err)
	}

	var matchingDynakubes []dynatracev1alpha1.DynaKube

	conflictCounter := ConflictCounter{}
	for _, dk := range dynakubes.Items {
		matches, err := nm.matchForDynakube(dk)
		if err != nil {
			return nil, err
		}
		if matches {
			if err := conflictCounter.Inc(&dk); err != nil {
				return nil, err
			}
			matchingDynakubes = append(matchingDynakubes, dk)
		}
	}
	nm.logger.Info("Matching dk found", "len(dynakubes)", len(matchingDynakubes))
	return matchingDynakubes, nil
}

func (nm NamespaceMapper) updateLabels(dynakubes []dynatracev1alpha1.DynaKube) {
	if nm.targetNs.Labels == nil {
		nm.targetNs.Labels = make(map[string]string)
	}
	if len(dynakubes) == 0 {
		nm.logger.Info("No matching dk found")
		delete(nm.targetNs.Labels, InstanceLabel)
	}
	processedDks := map[string]bool{}
	for i := range dynakubes {
		dk := &dynakubes[i]
		oldDkName, ok := nm.targetNs.Labels[InstanceLabel]
		if !ok || oldDkName != dk.Name {
			processedDks[dk.Name] = true
			addNamespaceInjectLabel(dk.Name, nm.targetNs)
		}
	}
}

func (nm NamespaceMapper) matchForDynakube(dk dynatracev1alpha1.DynaKube) (bool, error) {
	selector, err := metav1.LabelSelectorAsSelector(dk.Spec.MonitoredNamespaces)
	if err != nil {
		return false, errors.WithStack(err)
	}
	matches := selector.Matches(labels.Set(nm.targetNs.Labels))
	if matches {
		return matches, err
	}
	return matches, nil
}
