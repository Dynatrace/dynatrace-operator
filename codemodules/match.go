package codemodules

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtlabels"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func MatchForNamespace(codeModuleDynakubes []dynatracev1alpha1.DynaKube, namespace *corev1.Namespace) (*dynatracev1alpha1.DynaKube, error) {
	var matchingModules []dynatracev1alpha1.DynaKube

	for _, codeModule := range codeModuleDynakubes {
		matching, err := dtlabels.IsMatching(
			codeModule.Spec.CodeModules.Selector.MatchLabels,
			codeModule.Spec.CodeModules.Selector.MatchExpressions,
			namespace.Labels)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if matching {
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
