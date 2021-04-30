package codemodules

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func findCodeModules(ctx context.Context, clt client.Client) ([]dynatracev1alpha1.DynaKube, error) {
	dynaKubeList := &dynatracev1alpha1.DynaKubeList{}
	err := clt.List(ctx, dynaKubeList)
	if err != nil {
		return nil, errors.Cause(err)
	}

	var codeModules []dynatracev1alpha1.DynaKube
	for _, pod := range dynaKubeList.Items {
		if pod.Spec.CodeModules.Enabled {
			codeModules = append(codeModules, pod)
		}
	}

	return codeModules, nil
}

func matchForNamespace(codeModuleDynakubes []dynatracev1alpha1.DynaKube, namespace *corev1.Namespace) (*dynatracev1alpha1.DynaKube, error) {
	var matchingModules []dynatracev1alpha1.DynaKube

	for _, codeModule := range codeModuleDynakubes {
		selector, err := metav1.LabelSelectorAsSelector(&codeModule.Spec.CodeModules.Selector)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if selector.Matches(labels.Set(namespace.Labels)) {
			matchingModules = append(matchingModules, codeModule)
		}
	}

	if len(matchingModules) > 1 {
		return nil, errors.New("namespace matches two DynaKubes which is unsupported. " +
			"refine the labels on your namespace metadata or DynaKube/CodeModules specification")
	}
	if len(matchingModules) == 0 {
		return nil, nil
	}
	return &matchingModules[0], nil
}

func FindForNamespace(ctx context.Context, clt client.Client, namespace *corev1.Namespace) (*dynatracev1alpha1.DynaKube, error) {
	codeModules, err := findCodeModules(ctx, clt)
	if err != nil {
		return nil, err
	}

	return matchForNamespace(codeModules, namespace)
}
