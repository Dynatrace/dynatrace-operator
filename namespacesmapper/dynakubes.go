package namespacesmapper

import (
	"context"
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func updateMapFromDynaKube(dk *dynatracev1alpha1.DynaKube, selector labels.Selector, cfgmap *corev1.ConfigMap, nsList *corev1.NamespaceList) bool {
	updated := false

	if cfgmap.Data == nil {
		cfgmap.Data = make(map[string]string)
	}

	for _, namespace := range nsList.Items {
		matches := selector.Matches(labels.Set(namespace.Labels))
		mapdkname, ok := cfgmap.Data[namespace.Name]

		if matches {
			if !ok || mapdkname != dk.Name {
				cfgmap.Data[namespace.Name] = dk.Name
				updated = true
			}
		} else {
			if ok && mapdkname == dk.Name {
				delete(cfgmap.Data, namespace.Name)
				updated = true
			}
		}
	}
	return updated
}

func doMapFromDynaKube(ctx context.Context, clt client.Client, opns string, cfgmapname string, dk *dynatracev1alpha1.DynaKube, crSelector selectorFunc, nsList *corev1.NamespaceList) error {
	cfgmap, err := getOrCreateMap(ctx, clt, opns, cfgmapname)
	if err != nil {
		return err
	}

	selector, err := metav1.LabelSelectorAsSelector(crSelector(*dk))
	if err != nil {
		return errors.WithStack(err)
	}

	if updateMapFromDynaKube(dk, selector, cfgmap, nsList) {
		if err := clt.Update(ctx, cfgmap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgmapname)
		}
	}
	return nil
}

func MapFromDynaKube(ctx context.Context, clt client.Client, opns string, dk *dynatracev1alpha1.DynaKube) error {
	nsList := &corev1.NamespaceList{}
	err := clt.List(ctx, nsList)

	if err != nil {
		return errors.Cause(err)
	}

	diErr := doMapFromDynaKube(ctx, clt, opns, dataIngestMapName, dk,
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.DataIngestSpec.Selector
		}, nsList)

	cmErr := doMapFromDynaKube(ctx, clt, opns, codeModulesMapName, dk,
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.Selector
		}, nsList)

	if diErr != nil && cmErr != nil {
		return fmt.Errorf("%s ; %s", diErr.Error(), cmErr.Error())
	}
	if diErr != nil {
		return diErr
	}
	if cmErr != nil {
		return cmErr
	}
	return nil
}

func removeDynaKubeFromMap(cfgmap *corev1.ConfigMap, dkname string) bool {
	updated := false

	if cfgmap.Data == nil {
		cfgmap.Data = make(map[string]string)
	}

	for namespace, dynakube := range cfgmap.Data {
		if dynakube == dkname {
			delete(cfgmap.Data, namespace)
			updated = true
		}
	}
	return updated
}

func doUnmapFromDynaKube(ctx context.Context, clt client.Client, opns string, dkname string, mapName string) error {
	nsmap, err := getOrCreateMap(ctx, clt, opns, mapName)
	if err != nil {
		return err
	}

	if removeDynaKubeFromMap(nsmap, dkname) {
		if err := clt.Update(ctx, nsmap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", mapName)
		}
	}
	return nil
}

func UnmapFromDynaKube(ctx context.Context, clt client.Client, opns string, dkname string) error {
	diErr := doUnmapFromDynaKube(ctx, clt, opns, dkname, dataIngestMapName)
	cmErr := doUnmapFromDynaKube(ctx, clt, opns, dkname, codeModulesMapName)

	if diErr != nil && cmErr != nil {
		return fmt.Errorf("%s ; %s", diErr.Error(), cmErr.Error())
	}
	if diErr != nil {
		return diErr
	}
	if cmErr != nil {
		return cmErr
	}
	return nil
}
