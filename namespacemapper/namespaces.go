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

func MapFromNamespace(ctx context.Context, clt client.Client, operatorNs string, ns corev1.Namespace) error {
	err := mapFromNamespaceDataIngest(ctx, clt, operatorNs, ns)
	if err != nil {
		return err
	}
	return mapFromNamespaceCodeModules(ctx, clt, operatorNs, ns)
}

func UnmapFromNamespace(ctx context.Context, clt client.Client, operatorNs string, namespace string) error {
	err := unmapFromNamespace(ctx, clt, operatorNs, namespace, dataIngestMapName)
	if err != nil {
		return err
	}
	return unmapFromNamespace(ctx, clt, operatorNs, namespace, codeModulesMapName)
}

func mapFromNamespaceDataIngest(ctx context.Context, clt client.Client, operatorNs string, ns corev1.Namespace) error {
	return mapFromNamespace(ctx, clt, operatorNs, ns, dataIngestMapName,
		func(dk dynatracev1alpha1.DynaKube) bool {
			return dk.Spec.DataIngestSpec.Enabled
		},
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.DataIngestSpec.Selector
		})
}

func mapFromNamespaceCodeModules(ctx context.Context, clt client.Client, operatorNs string, ns corev1.Namespace) error {
	return mapFromNamespace(ctx, clt, operatorNs, ns, codeModulesMapName,
		func(dk dynatracev1alpha1.DynaKube) bool {
			return dk.Spec.CodeModules.Enabled
		},
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.Selector
		})
}

func findDynaKubes(ctx context.Context, clt client.Client, dynakubeFilter dynaKubeFilterFunc) ([]dynatracev1alpha1.DynaKube, error) {
	dynaKubeList := &dynatracev1alpha1.DynaKubeList{}
	err := clt.List(ctx, dynaKubeList)
	if err != nil {
		return nil, errors.Cause(err)
	}

	var dynakubes []dynatracev1alpha1.DynaKube
	for _, dynakube := range dynaKubeList.Items {
		if dynakubeFilter(dynakube) {
			dynakubes = append(dynakubes, dynakube)
		}
	}

	return dynakubes, nil
}

func matchForNamespace(dynakubes []dynatracev1alpha1.DynaKube, namespace *corev1.Namespace, nsSelector namespaceSelectorFunc) (*dynatracev1alpha1.DynaKube, error) {
	var matchingDynakubes []dynatracev1alpha1.DynaKube

	for _, dk := range dynakubes {
		selector, err := metav1.LabelSelectorAsSelector(nsSelector(dk))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if selector.Matches(labels.Set(namespace.Labels)) {
			matchingDynakubes = append(matchingDynakubes, dk)
		}
	}

	if len(matchingDynakubes) > 1 {
		return nil, errors.New("namespace matches two or more DynaKubes which is unsupported. " +
			"refine the labels on your namespace metadata or DynaKube/CodeModules specification")
	}
	if len(matchingDynakubes) == 0 {
		return nil, nil
	}
	return &matchingDynakubes[0], nil
}

func findForNamespace(ctx context.Context, clt client.Client, namespace *corev1.Namespace, dkFilter dynaKubeFilterFunc, nsSelector namespaceSelectorFunc) (*dynatracev1alpha1.DynaKube, error) {
	dynakubes, err := findDynaKubes(ctx, clt, dkFilter)

	if err != nil {
		return nil, err
	}

	return matchForNamespace(dynakubes, namespace, nsSelector)
}

func mapFromNamespace(ctx context.Context, clt client.Client, operatorNs string, ns corev1.Namespace, cfgMapName string, dkFilter dynaKubeFilterFunc, nsSelector namespaceSelectorFunc) error {
	dynakube, err := findForNamespace(ctx, clt, &ns, dkFilter, nsSelector)

	if err != nil {
		return err
	}

	cfgMap, err := getOrCreateMap(ctx, clt, operatorNs, cfgMapName)
	if err != nil {
		return err
	}

	if dynakube == nil {
		return removeFromMap(ctx, clt, cfgMapName, cfgMap, ns.Name)
	}
	return updateMap(ctx, clt, cfgMapName, cfgMap, ns.Name, dynakube.Name)
}

func unmapFromNamespace(ctx context.Context, clt client.Client, operatorNs string, namespace string, cfgMapName string) error {
	cfgMap, err := getOrCreateMap(ctx, clt, operatorNs, cfgMapName)
	if err != nil {
		return err
	}

	return removeFromMap(ctx, clt, cfgMapName, cfgMap, namespace)
}

// updateMap sets `dkName` value to `namespace` key in `cfgMap`
func updateMap(ctx context.Context, clt client.Client, cfgMapName string, cfgMap *corev1.ConfigMap, namespace string, dkName string) error {
	if cfgMap.Data == nil {
		cfgMap.Data = make(map[string]string)
	}
	if dk, ok := cfgMap.Data[namespace]; !ok || dk != dkName {
		cfgMap.Data[namespace] = dkName

		if err := clt.Update(ctx, cfgMap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgMapName)
		}
	}
	return nil
}

func removeFromMap(ctx context.Context, clt client.Client, cfgMapName string, cfgMap *corev1.ConfigMap, namespace string) error {
	if cfgMap.Data == nil {
		return nil
	}
	if _, ok := cfgMap.Data[namespace]; ok {
		delete(cfgMap.Data, namespace)

		if err := clt.Update(ctx, cfgMap); err != nil {
			return errors.WithMessagef(err, "failed to remove namespace %s from %s", namespace, cfgMapName)
		}
	}
	return nil
}
