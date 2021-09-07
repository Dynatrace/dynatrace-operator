package mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DynakubeMapper struct {
	ctx        context.Context
	client     client.Client
	apiReader  client.Reader
	operatorNs string
	dk         *dynatracev1alpha1.DynaKube
}

func NewDynakubeMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, dk *dynatracev1alpha1.DynaKube) DynakubeMapper {
	return DynakubeMapper{ctx, clt, apiReader, operatorNs, dk}
}

func (dm DynakubeMapper) MapFromDynakube() error {
	nsList := &corev1.NamespaceList{}
	if err := dm.apiReader.List(dm.ctx, nsList); err != nil {
		return errors.Cause(err)
	}

	selector, err := metav1.LabelSelectorAsSelector(dm.dk.Spec.MonitoredNamespaces)
	if err != nil {
		return errors.WithStack(err)
	}

	return dm.updateAnnotations(selector, nsList)
}

func (dm DynakubeMapper) UnmapFromDynaKube() error {
	keys := getAnnotationKeys()
	var nsList corev1.NamespaceList
	for _, key := range keys {
		nsList, err := GetNamespacesForDynakube(dm.ctx, key, dm.apiReader, dm.dk.Name)
		if err != nil {
			return errors.WithMessagef(err, "failed to list namespaces for dynakube %s", dm.dk.Name)
		}
		if len(nsList) != 0 {
			break
		}
	}
	for _, ns := range nsList.Items {
		if err := removeNamespaceAnnotation(dm.ctx, keys, dm.client, &ns); err != nil {
			return err
		}
	}
	return nil
}

func (dm DynakubeMapper) updateAnnotations(selector labels.Selector, nsList *corev1.NamespaceList) error {
	var updated bool
	var modifiedNs []corev1.Namespace
	for _, namespace := range nsList.Items {
		if dm.operatorNs == namespace.Name {
			continue
		}
		if namespace.Annotations == nil {
			namespace.Annotations = make(map[string]string)
		}
		matches := selector.Matches(labels.Set(namespace.Labels))
		for key, filter := range options {
			dynakubeName, ok := namespace.Annotations[key]
			if matches {
				if filter(dm.dk) && (!ok || dynakubeName != dm.dk.Name) {
					updated = true
					namespace.Annotations[key] = dm.dk.Name
					modifiedNs = append(modifiedNs, namespace)
				} else if !filter(dm.dk) && ok {
					updated = true
					delete(namespace.Annotations, key)
					modifiedNs = append(modifiedNs, namespace)
				}
			} else if ok && dynakubeName == dm.dk.Name {
				updated = true
				delete(namespace.Annotations, key)
				modifiedNs = append(modifiedNs, namespace)
			}
		}
	}
	if updated {
		for _, ns := range modifiedNs {
			if err := dm.client.Update(dm.ctx, &ns); err != nil {
				return err
			}
		}
	}
	return nil
}
