package mapper

import (
	"context"
	"regexp"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func GetDynakubeForNamespace(ctx context.Context, clt client.Reader, namespaceName string, namespaceLabels map[string]string) (*dynatracev1beta1.DynaKube, error) {
	deployedDynakubes := &dynatracev1beta1.DynaKubeList{}
	err := clt.List(ctx, deployedDynakubes)
	if err != nil {
		return nil, err
	}

	var dynakubeForNamespace *dynatracev1beta1.DynaKube

	for i := range deployedDynakubes.Items {
		dynakube := &deployedDynakubes.Items[i]
		if isIgnoredNamespace(dynakube, namespaceName) {
			continue
		}

		matches, err := match(dynakube, namespaceLabels)
		if err != nil {
			return nil, err
		}

		if matches {
			if dynakubeForNamespace != nil {
				return nil, errors.New(ErrorConflictingNamespace)
			}
			dynakubeForNamespace = dynakube
		}
	}
	return dynakubeForNamespace, nil
}

func GetNamespacesForDynakube(ctx context.Context, clt client.Reader, dkName string) ([]corev1.Namespace, error) {
	nsList := &corev1.NamespaceList{}
	listOps := []client.ListOption{
		client.MatchingLabels(map[string]string{dtwebhook.InjectionInstanceLabel: dkName}),
	}
	err := clt.List(ctx, nsList, listOps...)
	if err != nil {
		return nil, err
	}
	return nsList.Items, err
}

func GetSecrets(ctx context.Context, clt client.Reader, secretName string) ([]corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	listOps := client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", secretName),
	}
	err := clt.List(ctx, secretList, &listOps)
	if err != nil {
		return nil, err
	}
	return secretList.Items, err
}

func addNamespaceInjectLabel(dkName string, ns *corev1.Namespace) {
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels[dtwebhook.InjectionInstanceLabel] = dkName
}

func setUpdatedViaDynakubeAnnotation(ns *corev1.Namespace) {
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	ns.Annotations[UpdatedViaDynakubeAnnotation] = "true"
}

// match uses the namespace selector in the dynakube to check if it matches a given namespace
// if the namespace selector is not set on the dynakube its an automatic match
// match uses the namespace selector in the dynakube to check if it matches a given namespace
// if the namespace selector is not set on the dynakube its an automatic match
func match(dk *dynatracev1beta1.DynaKube, namespaceLabels map[string]string) (bool, error) {
	if dk.NamespaceSelector() == nil {
		return true, nil
	}

	selector, err := metav1.LabelSelectorAsSelector(dk.NamespaceSelector())
	if err != nil {
		return false, errors.WithStack(err)
	}

	return selector.Matches(labels.Set(namespaceLabels)), nil
}

// updateNamespace tries to match the namespace to every dynakube with codeModules
// finds conflicting dynakubes(2 dynakube with codeModules on the same namespace)
// adds/updates/removes labels from the namespace.
func updateNamespace(namespace *corev1.Namespace, deployedDynakubes *dynatracev1beta1.DynaKubeList) (bool, error) {
	namespaceUpdated := false
	conflict := ConflictChecker{}
	for i := range deployedDynakubes.Items {
		dynakube := &deployedDynakubes.Items[i]
		if isIgnoredNamespace(dynakube, namespace.Name) {
			continue
		}
		matches, err := match(dynakube, namespace.Labels)

		if err != nil {
			return namespaceUpdated, err
		}
		if matches {
			if err := conflict.check(dynakube); err != nil {
				return namespaceUpdated, err
			}
		}

		labelsUpdated := updateLabels(matches, dynakube, namespace)
		namespaceUpdated = labelsUpdated || namespaceUpdated
	}
	return namespaceUpdated, nil
}

func updateLabels(matches bool, dynakube *dynatracev1beta1.DynaKube, namespace *corev1.Namespace) bool {
	updated := false
	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}

	associatedDynakubeName, instanceLabelFound := namespace.Labels[dtwebhook.InjectionInstanceLabel]

	if matches && dynakube.NeedAppInjection() {
		if !instanceLabelFound || associatedDynakubeName != dynakube.Name {
			updated = true
			addNamespaceInjectLabel(dynakube.Name, namespace)
			log.Info("started monitoring namespace", "namespace", namespace.Name, "dynakube", dynakube.Name)
		}
	} else if instanceLabelFound && associatedDynakubeName == dynakube.Name {
		updated = true
		delete(namespace.Labels, dtwebhook.InjectionInstanceLabel)
	}
	return updated
}

func isIgnoredNamespace(dk *dynatracev1beta1.DynaKube, namespaceName string) bool {
	for _, pattern := range dk.FeatureIgnoredNamespaces() {
		if matched, _ := regexp.MatchString(pattern, namespaceName); matched {
			return true
		}
	}
	return false
}

func IsMatchingNamespace(namespaceName string, namespaceLabels map[string]string, dynakube *dynatracev1beta1.DynaKube) (bool, error) {
	if isIgnoredNamespace(dynakube, namespaceName) {
		return false, nil
	}

	matches, err := match(dynakube, namespaceLabels)
	if err != nil {
		return false, err
	}

	return matches, nil
}
