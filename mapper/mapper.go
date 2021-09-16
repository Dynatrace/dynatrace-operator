package mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InstanceLabel                = "dynakube.dynatrace.com/instance"
	UpdatedViaDynakubeAnnotation = "dynatrace.com/updated-via-operator"
)

type ConflictChecker struct {
	alreadyUsed bool
}

func (c *ConflictChecker) check(dk *dynatracev1alpha1.DynaKube) error {
	if !dk.Spec.CodeModules.Enabled {
		return nil
	}
	if c.alreadyUsed {
		return errors.New("namespace matches two or more DynaKubes which is unsupported. " +
			"refine the labels on your namespace metadata or DynaKube/CodeModules specification")
	}
	c.alreadyUsed = true
	return nil
}

func GetNamespacesForDynakube(ctx context.Context, clt client.Reader, dkName string) ([]corev1.Namespace, error) {
	nsList := &corev1.NamespaceList{}
	listOps := []client.ListOption{
		client.MatchingLabels(map[string]string{InstanceLabel: dkName}),
	}
	err := clt.List(ctx, nsList, listOps...)
	if err != nil {
		return nil, err
	}
	return nsList.Items, err
}

func addNamespaceInjectLabel(dkName string, ns *corev1.Namespace) {
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels[InstanceLabel] = dkName
}

func setUpdatedByDynakubeAnnotation(ns *corev1.Namespace) {
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	ns.Annotations[UpdatedViaDynakubeAnnotation] = "true"
}
