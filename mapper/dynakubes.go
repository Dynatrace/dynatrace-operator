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
	if err := dm.apiReader.List(dm.ctx, dkList, &client.ListOptions{Namespace: dm.operatorNs}); err != nil {
		return errors.Cause(err)
	}

	return dm.checkDynakubes(nsList, dkList)
}

func (dm DynakubeMapper) UnmapFromDynaKube() error {
	var nsList []corev1.Namespace
	var err error
	nsList, err = GetNamespacesForDynakube(dm.ctx, dm.apiReader, dm.dk.Name)
	if err != nil {
		return errors.WithMessagef(err, "failed to list namespaces for dynakube %s", dm.dk.Name)
	}
	for _, ns := range nsList {
		delete(ns.Labels, InstanceLabel)
		setUpdatedByDynakubeAnnotation(&ns)
		if err := dm.client.Update(dm.ctx, &ns); err != nil {
			return errors.WithMessagef(err, "failed to remove label %s from namespace %s", InstanceLabel, ns.Name)
		}
	}
	return nil
}

func (dm DynakubeMapper) checkDynakubes(nsList *corev1.NamespaceList, dkList *dynatracev1alpha1.DynaKubeList) error {
	var updated bool
	var err error
	var modifiedNs []corev1.Namespace
	for _, namespace := range nsList.Items {
		if dm.operatorNs == namespace.Name {
			continue
		}
		if namespace.Labels == nil {
			namespace.Labels = make(map[string]string)
		}
		processedDks := map[string]bool{}
		conflictCounter := ConflictCounter{}
		for _, dk := range dkList.Items {
			if dk.Name == dm.dk.Name {
				dk = *dm.dk
			}
			updated, namespace, err = dm.updateLabels(dk, namespace, processedDks)
			if err != nil {
				return err
			}
			if updated {
				if err := conflictCounter.Inc(&dk); err != nil {
					return err
				}
				modifiedNs = append(modifiedNs, namespace)
			}
		}

	}
	if updated {
		for _, ns := range modifiedNs {
			setUpdatedByDynakubeAnnotation(&ns)
			if err := dm.client.Update(dm.ctx, &ns); err != nil {
				return err
			}
		}
	}
	return err
}

func (dm DynakubeMapper) updateLabels(dk dynatracev1alpha1.DynaKube, namespace corev1.Namespace, processedDks map[string]bool) (bool, corev1.Namespace, error) {
	updated := false
	selector, err := metav1.LabelSelectorAsSelector(dk.Spec.MonitoredNamespaces)
	if err != nil {
		return false, corev1.Namespace{}, errors.WithStack(err)
	}
	matches := selector.Matches(labels.Set(namespace.Labels))
	oldDkName, ok := namespace.Labels[InstanceLabel]
	if matches {
		if !ok || oldDkName != dk.Name {
			updated = true
			addNamespaceInjectLabel(dk.Name, &namespace)
			processedDks[dk.Name] = true
		}
	} else if ok && oldDkName == dk.Name {
		updated = true
		delete(namespace.Labels, InstanceLabel)
	}
	return updated, namespace, nil
}
