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

	dkList := &dynatracev1alpha1.DynaKubeList{}
	if err := dm.apiReader.List(dm.ctx, dkList); err != nil {
		return errors.Cause(err)
	}

	return dm.checkDynakubes(nsList, dkList)
}

func (dm DynakubeMapper) UnmapFromDynaKube() error {
	keys := getAnnotationKeys()
	var nsList []*corev1.Namespace
	var err error
	for _, key := range keys {
		nsList, err = GetNamespacesForDynakube(dm.ctx, key, dm.apiReader, dm.dk.Name)
		if err != nil {
			return errors.WithMessagef(err, "failed to list namespaces for dynakube %s", dm.dk.Name)
		}
		if len(nsList) != 0 {
			break
		}
	}
	for _, ns := range nsList {
		removeNamespaceAnnotation(dm.ctx, keys, dm.client, ns)
		ns.Annotations[UpdatedByDynakube] = "true"
		if err := dm.client.Update(dm.ctx, ns); err != nil {
			return errors.WithMessagef(err, "failed to remove annotation %s from namespace %s", keys, ns.Name)
		}
	}
	return nil
}

func (dm DynakubeMapper) checkDynakubes(nsList *corev1.NamespaceList, dkList *dynatracev1alpha1.DynaKubeList) error {
	var updated bool
	var err error
	var modifiedNs []corev1.Namespace
	for _, namespace := range nsList.Items {
		filterCheck := conflictChecker{}
		if dm.operatorNs == namespace.Name {
			continue
		}
		if namespace.Annotations == nil {
			namespace.Annotations = make(map[string]string)
		}
		processedDks := map[string]bool{}
		for _, dk := range dkList.Items {
			if dk.Name == dm.dk.Name {
				dk = *dm.dk
			}
			updated, namespace, err = dm.updateAnnotations(dk, namespace, filterCheck, processedDks)
			if err != nil {
				return err
			}
			if updated {
				modifiedNs = append(modifiedNs, namespace)
			}
		}

	}
	if updated {
		for _, ns := range modifiedNs {
			ns.Annotations[UpdatedByDynakube] = "true"
			if err := dm.client.Update(dm.ctx, &ns); err != nil {
				return err
			}
		}
	}
	return err
}

func (dm DynakubeMapper) updateAnnotations(dk dynatracev1alpha1.DynaKube, namespace corev1.Namespace, filterCheck conflictChecker, processedDks map[string]bool) (bool, corev1.Namespace, error) {
	updated := false
	selector, err := metav1.LabelSelectorAsSelector(dk.Spec.MonitoredNamespaces)
	if err != nil {
		return false, corev1.Namespace{}, errors.WithStack(err)
	}
	matches := selector.Matches(labels.Set(namespace.Labels))
	for key, filter := range options {
		oldDkName, ok := namespace.Annotations[key]
		if matches {
			if filter(&dk) && (!ok || oldDkName != dk.Name) {
				if err := filterCheck.Inc(key); err != nil {
					return updated, namespace, err
				}
				updated = true
				namespace.Annotations[key] = dk.Name
				processedDks[dk.Name] = true
			} else if !filter(&dk) && ok && !processedDks[oldDkName] {
				updated = true
				delete(namespace.Annotations, key)
			}
		} else if ok && oldDkName == dk.Name {
			updated = true
			delete(namespace.Annotations, key)
		}
	}
	return updated, namespace, nil
}
