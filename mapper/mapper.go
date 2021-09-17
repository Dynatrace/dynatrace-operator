package mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

func match(dk *dynatracev1alpha1.DynaKube, namespace *corev1.Namespace) (bool, error) {
	matches := false
	if dk.Spec.MonitoredNamespaces == nil {
		matches = true
	} else {
		selector, err := metav1.LabelSelectorAsSelector(dk.Spec.MonitoredNamespaces)
		if err != nil {
			return matches, errors.WithStack(err)
		}
		matches = selector.Matches(labels.Set(namespace.Labels))
	}
	return matches, nil
}

// updateNamespace tries to match the namespace to every dynakube with codeModules
// finds conflicting dynakubes(2 dynakube with codeModules on the same namespace)
func updateNamespace(namespace *corev1.Namespace, dkList *dynatracev1alpha1.DynaKubeList) (bool, error) {
	var updated bool
	conflict := ConflictChecker{}
	for i := range dkList.Items {
		dynakube := &dkList.Items[i]
		matches, err := match(dynakube, namespace)
		if err != nil {
			return updated, err
		}
		if matches {
			if err := conflict.check(dynakube); err != nil {
				return updated, err
			}
		}
		upd, err := updateLabels(matches, dynakube, namespace)
		if err != nil {
			return updated, err
		}
		if upd {
			updated = true
		}
	}
	return updated, nil
}

func updateLabels(matches bool, dynakube *dynatracev1alpha1.DynaKube, namespace *corev1.Namespace) (bool, error) {
	updated := false
	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}
	oldDkName, ok := namespace.Labels[InstanceLabel]
	if matches && dynakube.Spec.CodeModules.Enabled {
		if !ok || oldDkName != dynakube.Name {
			updated = true
			addNamespaceInjectLabel(dynakube.Name, namespace)
		}
	} else if ok && oldDkName == dynakube.Name {
		updated = true
		delete(namespace.Labels, InstanceLabel)
	}
	return updated, nil
}
