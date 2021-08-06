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

func doMapFromNamespace(ctx context.Context, clt client.Client, opns string, ns corev1.Namespace, mapName string, dkSelector dynaKubeFunc, crSelector selectorFunc) error {
	dynakube, err := findForNamespace(ctx, clt, &ns, dkSelector, crSelector)

	if err != nil {
		return err
	}

	cfgmap, err := getOrCreateMap(ctx, clt, opns, mapName)
	if err != nil {
		return err
	}

	if dynakube == nil {
		return removeFromMap(ctx, clt, mapName, cfgmap, ns.Name)
	}
	return updateMap(ctx, clt, mapName, cfgmap, ns.Name, dynakube.Name)
}

func MapFromNamespace(ctx context.Context, clt client.Client, opns string, ns corev1.Namespace) error {
	diErr := doMapFromNamespace(ctx, clt, opns, ns, dataIngestMapName,
		func(dk dynatracev1alpha1.DynaKube) bool {
			return dk.Spec.DataIngestSpec.Enabled
		},
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.DataIngestSpec.Selector
		})

	cmErr := doMapFromNamespace(ctx, clt, opns, ns, codeModulesMapName,
		func(dk dynatracev1alpha1.DynaKube) bool {
			return dk.Spec.CodeModules.Enabled
		},
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.Selector
		})

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

func doUnmapFromNamespace(ctx context.Context, clt client.Client, opns string, namespace string, mapName string) error {
	nsmap, err := getOrCreateMap(ctx, clt, opns, mapName)
	if err != nil {
		return err
	}

	return removeFromMap(ctx, clt, mapName, nsmap, namespace)
}

func UnmapFromNamespace(ctx context.Context, clt client.Client, opns string, namespace string) error {
	diErr := doUnmapFromNamespace(ctx, clt, opns, namespace, dataIngestMapName)
	cmErr := doUnmapFromNamespace(ctx, clt, opns, namespace, codeModulesMapName)

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

func updateMap(ctx context.Context, clt client.Client, cfgmapname string, cfgmap *corev1.ConfigMap, namespace string, dkname string) error {
	if cfgmap.Data == nil {
		cfgmap.Data = make(map[string]string)
	}
	if dk, ok := cfgmap.Data[namespace]; !ok || dk != dkname {
		cfgmap.Data[namespace] = dkname

		if err := clt.Update(ctx, cfgmap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgmapname)
		}
	}
	return nil
}

func removeFromMap(ctx context.Context, clt client.Client, cfgmapname string, cfgmap *corev1.ConfigMap, namespace string) error {
	if cfgmap.Data == nil {
		return nil
	}
	if _, ok := cfgmap.Data[namespace]; ok {
		delete(cfgmap.Data, namespace)

		if err := clt.Update(ctx, cfgmap); err != nil {
			return errors.WithMessagef(err, "failed to remove namespace %s from %s", namespace, cfgmapname)
		}
	}
	return nil
}
