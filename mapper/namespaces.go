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
		keys := getAnnotationKeys()
		removeNamespaceAnnotation(nm.ctx, keys, nm.client, nm.targetNs)
		return nil
	}
	for i := range dynakubes {
		nm.updateAnnotations(&dynakubes[i])
	}
	return nil
}

func (nm NamespaceMapper) findDynakubesForNamespace() ([]dynatracev1alpha1.DynaKube, error) {
	dynakubes := &dynatracev1alpha1.DynaKubeList{}
	err := nm.client.List(nm.ctx, dynakubes)

	if err != nil {
		return nil, errors.Cause(err)
	}

	var matchingDynakubes []dynatracev1alpha1.DynaKube
	filterCheck := map[string]int{}

	for _, dk := range dynakubes.Items {
		selector, err := metav1.LabelSelectorAsSelector(dk.Spec.MonitoredNamespaces)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if selector.Matches(labels.Set(nm.targetNs.Labels)) {
			for key, filter := range options {
				if filter(&dk) {
					filterCheck[key] += 1
				}
			}
			matchingDynakubes = append(matchingDynakubes, dk)
		}
	}

	if len(matchingDynakubes) > 1 && hasDynakubeConflicts(filterCheck) {
		return nil, errors.New("namespace matches two or more DynaKubes which is unsupported. " +
			"refine the labels on your namespace metadata or DynaKube/CodeModules specification")
	}
	if len(matchingDynakubes) == 0 {
		nm.logger.Info("No matching dk found")
		return nil, nil
	}
	nm.logger.Info("Matching dk found", "dkName", matchingDynakubes[0].Name)
	return matchingDynakubes, nil
}

func (nm NamespaceMapper) updateAnnotations(dk *dynatracev1alpha1.DynaKube) {
	if nm.targetNs.Annotations == nil {
		nm.targetNs.Annotations = make(map[string]string)
	}
	for key, filter := range options {
		dkName, ok := nm.targetNs.Annotations[key]
		if filter(dk) && dkName != dk.Name {
			nm.targetNs.Annotations[key] = dk.Name
		} else if !filter(dk) && ok {
			delete(nm.targetNs.Annotations, key)
		}
	}
}

func hasDynakubeConflicts(filterChecks map[string]int) bool {
	for _, amount := range filterChecks {
		if amount > 1 {
			return true
		}
	}
	return false
}
