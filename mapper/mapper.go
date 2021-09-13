package mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConflictCounter struct {
	i int
}

func (c *ConflictCounter) Inc(dk *dynatracev1alpha1.DynaKube) error {
	if !dk.Spec.CodeModules.Enabled {
		return nil
	}
	c.i += 1
	if c.i > 1 {
		return errors.New("namespace matches two or more DynaKubes which is unsupported. " +
			"refine the labels on your namespace metadata or DynaKube/CodeModules specification")
	}
	return nil
}

const (
	InstanceLabel               = "dynakube.dynatrace.com/instance"
	UpdatedByDynakubeAnnotation = "dynatrace.com/updated-via-operator"
)

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

func removeNamespaceInjectLabel(ns *corev1.Namespace) {
	if ns.Labels == nil {
		return
	}
	delete(ns.Labels, InstanceLabel)
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
	ns.Annotations[UpdatedByDynakubeAnnotation] = "true"
}
