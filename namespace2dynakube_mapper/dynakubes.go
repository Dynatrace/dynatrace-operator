package namespace2dynakube_mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: dispatching between DataIngest and CodeModules should be more generic

func MapFromDynakube_DataIngest(ctx context.Context, clt client.Client, operatorNs string, dk *dynatracev1alpha1.DynaKube) error {
	nsList := &corev1.NamespaceList{}
	err := clt.List(ctx, nsList)

	if err != nil {
		return errors.Cause(err)
	}

	return doMapNs2Dk(ctx, clt, operatorNs, dataIngestMapName, dk, nsList, dk.Spec.DataIngestSpec.Selector)
}

func MapFromDynaKube_CodeModules(ctx context.Context, clt client.Client, operatorNs string, dk *dynatracev1alpha1.DynaKube) error {
	nsList := &corev1.NamespaceList{}
	err := clt.List(ctx, nsList)

	if err != nil {
		return errors.Cause(err)
	}

	return doMapNs2Dk(ctx, clt, operatorNs, codeModulesMapName, dk, nsList, dk.Spec.CodeModules.Selector)
}

func UnmapFromDynaKube(ctx context.Context, clt client.Client, operatorNs string, dkName string) error {
	err := doUnmapFromDynaKube(ctx, clt, operatorNs, dkName, dataIngestMapName)
	if err != nil {
		return err
	}
	err = doUnmapFromDynaKube(ctx, clt, operatorNs, dkName, codeModulesMapName)
	return err
}

// updateMapNs2Dk fills `cfgmap` (namespace: dynakube)
// with namespaces matching the selector(key) and corresponding dynakube name(value)
func updateMapNs2Dk(dk *dynatracev1alpha1.DynaKube, selector labels.Selector, cfgmap *corev1.ConfigMap, nsList *corev1.NamespaceList) bool {
	updated := false

	if cfgmap.Data == nil {
		cfgmap.Data = make(map[string]string)
	}

	for _, namespace := range nsList.Items {
		matches := selector.Matches(labels.Set(namespace.Labels))
		dynakubeName, ok := cfgmap.Data[namespace.Name]

		if matches {
			if !ok || dynakubeName != dk.Name {
				cfgmap.Data[namespace.Name] = dk.Name
				updated = true
			}
		} else {
			if ok && dynakubeName == dk.Name {
				delete(cfgmap.Data, namespace.Name)
				updated = true
			}
		}
	}
	return updated
}

func doMapNs2Dk(ctx context.Context, clt client.Client, operatorNs string, cfgmapName string, dk *dynatracev1alpha1.DynaKube, nsList *corev1.NamespaceList, ps *metav1.LabelSelector) error {
	cfgmap, err := getOrCreateMap(ctx, clt, operatorNs, cfgmapName)
	if err != nil {
		return err
	}

	selector, err := metav1.LabelSelectorAsSelector(ps)
	if err != nil {
		return errors.WithStack(err)
	}

	if updateMapNs2Dk(dk, selector, cfgmap, nsList) {
		if err := clt.Update(ctx, cfgmap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgmapName)
		}
	}
	return nil
}

func removeDkFromMap(cfgmap *corev1.ConfigMap, dkName string) bool {
	updated := false

	if cfgmap.Data == nil {
		cfgmap.Data = make(map[string]string)
	}

	for namespace, dynakube := range cfgmap.Data {
		if dynakube == dkName {
			delete(cfgmap.Data, namespace)
			updated = true
		}
	}
	return updated
}

func doUnmapFromDynaKube(ctx context.Context, clt client.Client, operatorNs string, dkName string, mapName string) error {
	nsmap, err := getOrCreateMap(ctx, clt, operatorNs, mapName)
	if err != nil {
		return err
	}

	if removeDkFromMap(nsmap, dkName) {
		if err := clt.Update(ctx, nsmap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", mapName)
		}
	}
	return nil
}
