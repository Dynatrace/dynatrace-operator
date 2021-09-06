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

func (dm DynakubeMapper) MapFromDynakubeDataIngest() error {
	nsList := &corev1.NamespaceList{}
	err := dm.client.List(dm.ctx, nsList)

	if err != nil {
		return errors.Cause(err)
	}

	return dm.createMapping(DataIngestMapName, nsList, dm.dk.Spec.DataIngestSpec.Selector)
}

func (dm DynakubeMapper) MapFromDynaKubeCodeModules() error {
	nsList := &corev1.NamespaceList{}
	err := dm.client.List(dm.ctx, nsList)

	if err != nil {
		return errors.Cause(err)
	}

	return dm.createMapping(CodeModulesMapName, nsList, dm.dk.Spec.CodeModules.NamespaceSelector)
}

func (dm DynakubeMapper) UnmapFromDynaKube() error {
	err := dm.unmapFromDynaKube(DataIngestMapName)
	if err != nil {
		return err
	}
	err = dm.unmapFromDynaKube(CodeModulesMapName)
	return err
}

// updateMapping fills `cfgMap` (namespace: dynakube)
// with namespaces matching the selector(key) and corresponding dynakube name(value)
func (dm DynakubeMapper) updateMapping(selector labels.Selector, cfgMap *corev1.ConfigMap, nsList *corev1.NamespaceList) (updated bool, modifiedNs, removedNs []corev1.Namespace) {
	if cfgMap.Data == nil {
		cfgMap.Data = make(map[string]string)
	}

	for _, namespace := range nsList.Items {
		matches := selector.Matches(labels.Set(namespace.Labels))
		dynakubeName, ok := cfgMap.Data[namespace.Name]

		if matches {
			if !ok || dynakubeName != dm.dk.Name {
				cfgMap.Data[namespace.Name] = dm.dk.Name
				updated = true
				modifiedNs = append(modifiedNs, namespace)
			}
		} else {
			if ok && dynakubeName == dm.dk.Name {
				delete(cfgMap.Data, namespace.Name)
				updated = true
				removedNs = append(removedNs, namespace)
			}
		}
	}
	return
}

func (dm DynakubeMapper) createMapping(cfgMapName string, nsList *corev1.NamespaceList, ps *metav1.LabelSelector) error {
	cfgMap, err := getOrCreateMap(dm.ctx, dm.client, dm.apiReader, dm.operatorNs, cfgMapName)
	if err != nil {
		return err
	}

	selector, err := metav1.LabelSelectorAsSelector(ps)
	if err != nil {
		return errors.WithStack(err)
	}

	updated, modifiedNs, removedNs := dm.updateMapping(selector, cfgMap, nsList)
	if updated {
		for _, ns := range modifiedNs {
			updateNamespaceLabel(dm.ctx, dm.client, dm.operatorNs, &ns, dm.dk)
			if err := dm.client.Update(dm.ctx, &ns); err != nil {
				return errors.WithMessagef(err, "failed to update namespace %s", ns.Name)
			}
		}
		for _, ns := range removedNs {
			removeNamespaceLabel(dm.ctx, dm.client, &ns)
			if err := dm.client.Update(dm.ctx, &ns); err != nil {
				return errors.WithMessagef(err, "failed to update namespace %s", ns.Name)
			}
		}
		if err := dm.client.Update(dm.ctx, cfgMap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgMapName)
		}
	}
	return nil
}

func (dm DynakubeMapper) removeDynaKubeFromMap(cfgMap *corev1.ConfigMap) bool {
	updated := false

	if cfgMap.Data == nil {
		cfgMap.Data = make(map[string]string)
	}

	for namespace, dynakube := range cfgMap.Data {
		if dynakube == dm.dk.Name {
			delete(cfgMap.Data, namespace)
			updated = true
		}
	}
	return updated
}

func (dm DynakubeMapper) unmapFromDynaKube(cfgMapName string) error {
	cfgMap, err := getOrCreateMap(dm.ctx, dm.client, dm.apiReader, dm.operatorNs, cfgMapName)
	if err != nil {
		return err
	}
	if dm.removeDynaKubeFromMap(cfgMap) {
		if err := dm.client.Update(dm.ctx, cfgMap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgMapName)
		}
		if cfgMapName == CodeModulesMapName {
			nsList, err := getNamespaceForDynakube(dm.ctx, dm.client, dm.dk.Name)
			if err != nil {
				return errors.WithMessagef(err, "failed to list namespaces for dynakube %s", dm.dk.Name)
			}
			for _, ns := range nsList.Items {
				removeNamespaceLabel(dm.ctx, dm.client, &ns)
				if err = dm.client.Update(dm.ctx, &ns); err != nil {
					return errors.WithMessagef(err, "failed to remove label from namespace %s", ns.Name)
				}
			}
		}
	}
	return nil
}

func getNamespaceForDynakube(ctx context.Context, clt client.Client, dkName string) (*corev1.NamespaceList, error) {
	nsList := &corev1.NamespaceList{}
	listOps := []client.ListOption{
		client.MatchingLabels(map[string]string{ReadyLabelKey: dkName}),
	}
	err := clt.List(ctx, nsList, listOps...)
	return nsList, err
}
