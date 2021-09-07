package mapper

import (
	"context"
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type dynaKubeFilterFunc func(dk *dynatracev1alpha1.DynaKube) bool

const (
	CodeModulesAnnotation = "dynatrace.com/dynakube-cm"
	DataIngestAnnotation  = "dynatrace.com/dynakube-di"
)

var options = map[string]dynaKubeFilterFunc{
	DataIngestAnnotation: func(dk *dynatracev1alpha1.DynaKube) bool {
		return dk.Spec.DataIngestSpec.Enabled
	},
	CodeModulesAnnotation: func(dk *dynatracev1alpha1.DynaKube) bool {
		return dk.Spec.CodeModules.Enabled
	},
}

func GetNamespaceForDynakube(ctx context.Context, annotationKey string, clt client.Reader, dkName string) (*corev1.NamespaceList, error) {
	nsList := &corev1.NamespaceList{}
	listOps := []client.ListOption{
		client.MatchingFields(map[string]string{fmt.Sprintf("metadata.annotations.%s", annotationKey): dkName}), // TODO
	}
	err := clt.List(ctx, nsList, listOps...)
	return nsList, err
}

func getAnnotationKeys() []string {
	keys := []string{}
	for key := range options {
		keys = append(keys, key)
	}
	return keys
}

func removeNamespaceAnnotation(ctx context.Context, annotationKeys []string, clt client.Client, ns *corev1.Namespace) error {
	if ns.Annotations == nil {
		return nil
	}
	for _, key := range annotationKeys {
		if _, ok := ns.Annotations[key]; !ok {
			return nil
		}
		delete(ns.Annotations, key)
	}
	if err := clt.Update(ctx, ns); err != nil {
		return errors.WithMessagef(err, "failed to remove annotation from namespace %s", annotationKeys, ns.Name)
	}
	return nil
}
