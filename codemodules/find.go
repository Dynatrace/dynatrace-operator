package codemodules

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FindCodeModules(ctx context.Context, clt client.Client) ([]dynatracev1alpha1.DynaKube, error) {
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
