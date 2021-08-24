package namespacesmapper

import (
	"context"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MapFromDynaKube_DI(ctx context.Context, clt client.Client, operatorNs string, dk *dynatracev1alpha1.DynaKube) error {
	nsList := &corev1.NamespaceList{}
	err := clt.List(ctx, nsList)

	if err != nil {
		return errors.Cause(err)
	}

	return doMapFromDynaKube(ctx, clt, operatorNs, dataIngestMapName, dk, nsList, dk.Spec.DataIngestSpec.Selector)
}

func MapFromDynaKube_CM(ctx context.Context, clt client.Client, operatorNs string, dk *dynatracev1alpha1.DynaKube) error {
	nsList := &corev1.NamespaceList{}
	err := clt.List(ctx, nsList)

	if err != nil {
		return errors.Cause(err)
	}

	return doMapFromDynaKube(ctx, clt, operatorNs, codeModulesMapName, dk, nsList, dk.Spec.CodeModules.Selector)
}

func UnmapFromDynaKube_DI(ctx context.Context, clt client.Client, operatorNs string, dkName string) error {
	return doUnmapFromDynaKube(ctx, clt, operatorNs, dkName, dataIngestMapName)
}
func UnmapFromDynaKube_CM(ctx context.Context, clt client.Client, operatorNs string, dkName string) error {
	return doUnmapFromDynaKube(ctx, clt, operatorNs, dkName, codeModulesMapName)
}

func updateMapFromDynaKube(dk *dynatracev1alpha1.DynaKube, selector labels.Selector, cfgmap *corev1.ConfigMap, nsList *corev1.NamespaceList) bool {
	updated := false

	if cfgmap.Data == nil {
		cfgmap.Data = make(map[string]string)
	}

	for _, namespace := range nsList.Items {
		matches := selector.Matches(labels.Set(namespace.Labels))
		mapDkName, ok := cfgmap.Data[namespace.Name]

		if matches {
			if !ok || mapDkName != dk.Name {
				cfgmap.Data[namespace.Name] = dk.Name
				updated = true
			}
		} else {
			if ok && mapDkName == dk.Name {
				delete(cfgmap.Data, namespace.Name)
				updated = true
			}
		}
	}
	return updated
}

func doMapFromDynaKube(ctx context.Context, clt client.Client, operatorNs string, cfgmapName string, dk *dynatracev1alpha1.DynaKube, nsList *corev1.NamespaceList, ps *metav1.LabelSelector) error {
	cfgmap, err := getOrCreateMap(ctx, clt, operatorNs, cfgmapName)
	if err != nil {
		return err
	}

	selector, err := metav1.LabelSelectorAsSelector(ps)
	if err != nil {
		return errors.WithStack(err)
	}

	if updateMapFromDynaKube(dk, selector, cfgmap, nsList) {
		if err := clt.Update(ctx, cfgmap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgmapName)
		}
	}
	return nil
}

func removeDynaKubeFromMap(cfgmap *corev1.ConfigMap, dkName string) bool {
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

	if removeDynaKubeFromMap(nsmap, dkName) {
		if err := clt.Update(ctx, nsmap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", mapName)
		}
	}
	return nil
}
