package namespacemapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MapFromDynakubeDataIngest(ctx context.Context, clt client.Client, operatorNs string, dk *dynatracev1alpha1.DynaKube) error {
	nsList := &corev1.NamespaceList{}
	err := clt.List(ctx, nsList)

	if err != nil {
		return errors.Cause(err)
	}

	return createMapping(ctx, clt, operatorNs, DataIngestMapName, dk, nsList, dk.Spec.DataIngestSpec.Selector)
}

func MapFromDynaKubeCodeModules(ctx context.Context, clt client.Client, operatorNs string, dk *dynatracev1alpha1.DynaKube) error {
	nsList := &corev1.NamespaceList{}
	err := clt.List(ctx, nsList)

	if err != nil {
		return errors.Cause(err)
	}

	return createMapping(ctx, clt, operatorNs, CodeModulesMapName, dk, nsList, dk.Spec.CodeModules.Selector)
}

func UnmapFromDynaKube(ctx context.Context, clt client.Client, operatorNs string, dkName string) error {
	err := unmapFromDynaKube(ctx, clt, operatorNs, dkName, DataIngestMapName)
	if err != nil {
		return err
	}
	err = unmapFromDynaKube(ctx, clt, operatorNs, dkName, CodeModulesMapName)
	return err
}

// updateMapping fills `cfgMap` (namespace: dynakube)
// with namespaces matching the selector(key) and corresponding dynakube name(value)
func updateMapping(dk *dynatracev1alpha1.DynaKube, selector labels.Selector, cfgMap *corev1.ConfigMap, nsList *corev1.NamespaceList) bool {
	updated := false

	if cfgMap.Data == nil {
		cfgMap.Data = make(map[string]string)
	}

	for _, namespace := range nsList.Items {
		matches := selector.Matches(labels.Set(namespace.Labels))
		dynakubeName, ok := cfgMap.Data[namespace.Name]

		if matches {
			if !ok || dynakubeName != dk.Name {
				cfgMap.Data[namespace.Name] = dk.Name
				updated = true
			}
		} else {
			if ok && dynakubeName == dk.Name {
				delete(cfgMap.Data, namespace.Name)
				updated = true
			}
		}
	}
	return updated
}

func createMapping(ctx context.Context, clt client.Client, operatorNs string, cfgMapName string, dk *dynatracev1alpha1.DynaKube, nsList *corev1.NamespaceList, ps *metav1.LabelSelector) error {
	cfgMap, err := getOrCreateMap(ctx, clt, operatorNs, cfgMapName)
	if err != nil {
		return err
	}

	selector, err := metav1.LabelSelectorAsSelector(ps)
	if err != nil {
		return errors.WithStack(err)
	}

	if updateMapping(dk, selector, cfgMap, nsList) {
		if err := clt.Update(ctx, cfgMap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgMapName)
		}
	}
	return nil
}

func removeDynaKubeFromMap(cfgMap *corev1.ConfigMap, dkName string) bool {
	updated := false

	if cfgMap.Data == nil {
		cfgMap.Data = make(map[string]string)
	}

	for namespace, dynakube := range cfgMap.Data {
		if dynakube == dkName {
			delete(cfgMap.Data, namespace)
			updated = true
		}
	}
	return updated
}

func unmapFromDynaKube(ctx context.Context, clt client.Client, operatorNs string, dkName string, cfgMapName string) error {
	cfgMap, err := getOrCreateMap(ctx, clt, operatorNs, cfgMapName)
	if err != nil {
		return err
	}

	if removeDynaKubeFromMap(cfgMap, dkName) {
		if err := clt.Update(ctx, cfgMap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgMapName)
		}
	}
	return nil
}
