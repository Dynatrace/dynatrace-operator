package mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type dynaKubeFilterFunc func(dk *dynatracev1alpha1.DynaKube) bool

type conflictChecker map[string]int

func (f conflictChecker) Inc(key string) error {
	f[key] += 1
	if f[key] > 1 {
		return errors.New("namespace matches two or more DynaKubes which is unsupported. " +
			"refine the labels on your namespace metadata or DynaKube/CodeModules specification")
	}
	return nil
}

const (
	CodeModulesAnnotation = "dynatrace.com/dynakube-cm"
	DataIngestAnnotation  = "dynatrace.com/dynakube-di"
	UpdatedByDynakube     = "dynatrace.com/dynakube-upd"
)

var options = map[string]dynaKubeFilterFunc{
	DataIngestAnnotation: func(dk *dynatracev1alpha1.DynaKube) bool {
		return dk.Spec.DataIngestSpec.Enabled
	},
	CodeModulesAnnotation: func(dk *dynatracev1alpha1.DynaKube) bool {
		return dk.Spec.CodeModules.Enabled
	},
}

func GetNamespacesForDynakube(ctx context.Context, annotationKey string, clt client.Reader, dkName string) ([]*corev1.Namespace, error) {
	nsList := &corev1.NamespaceList{}
	filteredNamespaces := []*corev1.Namespace{}
	err := clt.List(ctx, nsList)
	if err != nil {
		return nil, err
	}
	for i := range nsList.Items {
		if name := nsList.Items[i].Annotations[annotationKey]; dkName == name {
			filteredNamespaces = append(filteredNamespaces, &nsList.Items[i])
		}
	}
	return filteredNamespaces, err
}

func getAnnotationKeys() []string {
	keys := []string{}
	for key := range options {
		keys = append(keys, key)
	}
	return keys
}
func removeNamespaceAnnotation(ctx context.Context, annotationKeys []string, clt client.Client, ns *corev1.Namespace) {
	if ns.Annotations == nil {
		return
	}
	for _, key := range annotationKeys {
		if _, ok := ns.Annotations[key]; !ok {
			return
		}
		delete(ns.Annotations, key)
	}
}
