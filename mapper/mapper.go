package mapper

import (
	"context"
	"regexp"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InstanceLabel                = dtwebhook.LabelInstance
	UpdatedViaDynakubeAnnotation = "dynatrace.com/updated-via-operator"
	ErrorConflictingNamespace    = "namespace matches two or more DynaKubes which is unsupported. " +
		"refine the labels on your namespace metadata or DynaKube/CodeModules specification"
)

type ConflictChecker struct {
	alreadyUsed bool
}

func (c *ConflictChecker) check(dk *dynatracev1beta1.DynaKube) error {
	if !dk.NeedAppInjection() {
		return nil
	}
	if c.alreadyUsed {
		return errors.New(ErrorConflictingNamespace)
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

func setUpdatedViaDynakubeAnnotation(ns *corev1.Namespace) {
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	ns.Annotations[UpdatedViaDynakubeAnnotation] = "true"
}

// match uses the namespace selector in the dynakube to check if it matches a given namespace
// if the namspace selector is not set on the dynakube its an automatic match
func match(dk *dynatracev1beta1.DynaKube, namespace *corev1.Namespace) (bool, error) {
	matches := false
	if dk.NamespaceSelector() == nil {
		matches = true
	} else {
		selector, err := metav1.LabelSelectorAsSelector(dk.NamespaceSelector())
		if err != nil {
			return matches, errors.WithStack(err)
		}
		matches = selector.Matches(labels.Set(namespace.Labels))
	}
	return matches, nil
}

// updateNamespace tries to match the namespace to every dynakube with codeModules
// finds conflicting dynakubes(2 dynakube with codeModules on the same namespace)
// adds/updates/removes labels from the namespace.
func updateNamespace(operatorNs string, namespace *corev1.Namespace, dkList *dynatracev1beta1.DynaKubeList, log logr.Logger) (bool, error) {
	var updated bool
	conflict := ConflictChecker{}
	for i := range dkList.Items {
		dynakube := &dkList.Items[i]
		if operatorNs == namespace.Name || isIgnoredNamespace(dynakube, namespace.Name) {
			return false, nil
		}
		matches, err := match(dynakube, namespace)
		if err != nil {
			return updated, err
		}
		if matches {
			if err := conflict.check(dynakube); err != nil {
				return updated, err
			}
		}

		upd, err := updateLabels(matches, dynakube, namespace, log)
		if err != nil {
			return updated, err
		}
		if upd {
			updated = true
		}
	}
	return updated, nil
}

func updateLabels(matches bool, dynakube *dynatracev1beta1.DynaKube, namespace *corev1.Namespace, log logr.Logger) (bool, error) {
	updated := false
	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}
	oldDkName, ok := namespace.Labels[InstanceLabel]
	if matches && dynakube.NeedAppInjection() {
		if !ok || oldDkName != dynakube.Name {
			updated = true
			addNamespaceInjectLabel(dynakube.Name, namespace)
			log.Info("started monitoring namespace", "namespace", namespace.Name, "dynakube", dynakube.Name)
		}
	} else if ok && oldDkName == dynakube.Name {
		updated = true
		delete(namespace.Labels, InstanceLabel)
	}
	return updated, nil
}

func isIgnoredNamespace(dk *dynatracev1beta1.DynaKube, namespaceName string) bool {
	for _, pattern := range dk.FeatureIgnoredNamespaces() {
		if matched, _ := regexp.MatchString(pattern, namespaceName); matched {
			return true
		}
	}
	return false
}
