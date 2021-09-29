package mapper

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DynakubeMapper manages the mapping creation from the dynakube's side
type DynakubeMapper struct {
	ctx        context.Context
	client     client.Client
	apiReader  client.Reader
	operatorNs string
	dk         *dynatracev1beta1.DynaKube
}

func NewDynakubeMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, dk *dynatracev1beta1.DynaKube) DynakubeMapper {
	return DynakubeMapper{ctx, clt, apiReader, operatorNs, dk}
}

// MapFromDynakube checks all the namespaces to all the dynakubes
// updates the labels on the namespaces if necessary,
// finds confliction dynakubes (2 dynakube with codeModules on the same namespace)
func (dm DynakubeMapper) MapFromDynakube() error {
	nsList := &corev1.NamespaceList{}
	if err := dm.apiReader.List(dm.ctx, nsList); err != nil {
		return errors.Cause(err)
	}

	dkList := &dynatracev1beta1.DynaKubeList{}
	if err := dm.apiReader.List(dm.ctx, dkList, &client.ListOptions{Namespace: dm.operatorNs}); err != nil {
		return errors.Cause(err)
	}

	return dm.mapFromDynakube(nsList, dkList)
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
		setUpdatedViaDynakubeAnnotation(&ns)
		if err := dm.client.Update(dm.ctx, &ns); err != nil {
			return errors.WithMessagef(err, "failed to remove label %s from namespace %s", InstanceLabel, ns.Name)
		}
	}
	return nil
}

func (dm DynakubeMapper) mapFromDynakube(nsList *corev1.NamespaceList, dkList *dynatracev1beta1.DynaKubeList) error {
	var updated bool
	var err error
	var modifiedNs []*corev1.Namespace

	for i := range dkList.Items {
		if dkList.Items[i].Name == dm.dk.Name {
			dkList.Items[i] = *dm.dk
			break
		}
	}

	for i := range nsList.Items {
		namespace := &nsList.Items[i]
		if dm.operatorNs == namespace.Name {
			continue
		}
		updated, err = updateNamespace(namespace, dkList)
		if updated {
			modifiedNs = append(modifiedNs, namespace)
		}

	}
	for _, ns := range modifiedNs {
		setUpdatedViaDynakubeAnnotation(ns)
		if err := dm.client.Update(dm.ctx, ns); err != nil {
			return err
		}
	}
	return err
}
