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

func MapFromNamespace(ctx context.Context, clt client.Client, operatorNs string, ns corev1.Namespace) error {
	err := mapFromNamespace_DataIngest(ctx, clt, operatorNs, ns)
	if err != nil {
		return err
	}
	return mapFromNamespace_CodeModules(ctx, clt, operatorNs, ns)
}

func UnmapFromNamespace(ctx context.Context, clt client.Client, operatorNs string, namespace string) error {
	err := doUnmapFromNamespace(ctx, clt, operatorNs, namespace, dataIngestMapName)
	if err != nil {
		return err
	}
	return doUnmapFromNamespace(ctx, clt, operatorNs, namespace, codeModulesMapName)
}

func mapFromNamespace_DataIngest(ctx context.Context, clt client.Client, operatorNs string, ns corev1.Namespace) error {
	return doMapFromNamespace(ctx, clt, operatorNs, ns, dataIngestMapName,
		func(dk dynatracev1alpha1.DynaKube) bool {
			return dk.Spec.DataIngestSpec.Enabled
		},
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.DataIngestSpec.Selector
		})
}

func mapFromNamespace_CodeModules(ctx context.Context, clt client.Client, operatorNs string, ns corev1.Namespace) error {
	return doMapFromNamespace(ctx, clt, operatorNs, ns, codeModulesMapName,
		func(dk dynatracev1alpha1.DynaKube) bool {
			return dk.Spec.CodeModules.Enabled
		},
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.Selector
		})
}

func findDynaKubes(ctx context.Context, clt client.Client, dynakubeSelector dynaKubeFunc) ([]dynatracev1alpha1.DynaKube, error) {
	dynaKubeList := &dynatracev1alpha1.DynaKubeList{}
	err := clt.List(ctx, dynaKubeList)
	if err != nil {
		return nil, errors.Cause(err)
	}

	var dynakubes []dynatracev1alpha1.DynaKube
	for _, dynakube := range dynaKubeList.Items {
		if dynakubeSelector(dynakube) {
			dynakubes = append(dynakubes, dynakube)
		}
	}

	return dynakubes, nil
}

func matchForNamespace(dynakubes []dynatracev1alpha1.DynaKube, namespace *corev1.Namespace, selector selectorFunc) (*dynatracev1alpha1.DynaKube, error) {
	var matchingModules []dynatracev1alpha1.DynaKube

	for _, module := range dynakubes {
		selector, err := metav1.LabelSelectorAsSelector(selector(module))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if selector.Matches(labels.Set(namespace.Labels)) {
			matchingModules = append(matchingModules, module)
		}
	}

	if len(matchingModules) > 1 {
		return nil, errors.New("namespace matches two or more DynaKubes which is unsupported. " +
			"refine the labels on your namespace metadata or DynaKube/CodeModules specification")
	}
	if len(matchingModules) == 0 {
		return nil, nil
	}
	return &matchingModules[0], nil
}

func findForNamespace(ctx context.Context, clt client.Client, namespace *corev1.Namespace, dkSelector dynaKubeFunc, crSelector selectorFunc) (*dynatracev1alpha1.DynaKube, error) {
	dynakubes, err := findDynaKubes(ctx, clt, dkSelector)

	if err != nil {
		return nil, err
	}

	return matchForNamespace(dynakubes, namespace, crSelector)
}

func doMapFromNamespace(ctx context.Context, clt client.Client, operatorNs string, ns corev1.Namespace, mapName string, dkSelector dynaKubeFunc, crSelector selectorFunc) error {
	dynakube, err := findForNamespace(ctx, clt, &ns, dkSelector, crSelector)

	if err != nil {
		return err
	}

	cfgmap, err := getOrCreateMap(ctx, clt, operatorNs, mapName)
	if err != nil {
		return err
	}

	if dynakube == nil {
		return removeFromMap(ctx, clt, mapName, cfgmap, ns.Name)
	}
	return updateMap(ctx, clt, mapName, cfgmap, ns.Name, dynakube.Name)
}

func doUnmapFromNamespace(ctx context.Context, clt client.Client, operatorNs string, namespace string, mapName string) error {
	nsmap, err := getOrCreateMap(ctx, clt, operatorNs, mapName)
	if err != nil {
		return err
	}

	return removeFromMap(ctx, clt, mapName, nsmap, namespace)
}

// updateMap sets `dkname` value to `namespace` key in `cfgmap`
func updateMap(ctx context.Context, clt client.Client, cfgmapName string, cfgmap *corev1.ConfigMap, namespace string, dkname string) error {
	if cfgmap.Data == nil {
		cfgmap.Data = make(map[string]string)
	}
	if dk, ok := cfgmap.Data[namespace]; !ok || dk != dkname {
		cfgmap.Data[namespace] = dkname

		if err := clt.Update(ctx, cfgmap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgmapName)
		}
	}
	return nil
}

func removeFromMap(ctx context.Context, clt client.Client, cfgmapName string, cfgmap *corev1.ConfigMap, namespace string) error {
	if cfgmap.Data == nil {
		return nil
	}
	if _, ok := cfgmap.Data[namespace]; ok {
		delete(cfgmap.Data, namespace)

		if err := clt.Update(ctx, cfgmap); err != nil {
			return errors.WithMessagef(err, "failed to remove namespace %s from %s", namespace, cfgmapName)
		}
	}
	return nil
}
