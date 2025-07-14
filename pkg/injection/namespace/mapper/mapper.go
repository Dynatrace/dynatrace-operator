package mapper

import (
	"context"
	"regexp"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConflictChecker struct {
	alreadyUsed bool
}

func (c *ConflictChecker) check(dk *dynakube.DynaKube) error {
	if !dk.OneAgent().IsAppInjectionNeeded() && !dk.MetadataEnrichmentEnabled() {
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
		client.MatchingLabels(map[string]string{dtwebhook.InjectionInstanceLabel: dkName}),
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

	ns.Labels[dtwebhook.InjectionInstanceLabel] = dkName
}

func setUpdatedViaDynakubeAnnotation(ns *corev1.Namespace) {
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}

	ns.Annotations[UpdatedViaDynakubeAnnotation] = "true"
}

func match(dk *dynakube.DynaKube, namespace *corev1.Namespace) (bool, error) {
	if isIgnoredNamespace(dk, namespace.Name) {
		return false, nil
	}

	matchesOneAgent, err := matchOneAgent(dk, namespace)
	if err != nil {
		return false, err
	}

	matchesMetadataEnrichment, err := matchMetadataEnrichment(dk, namespace)
	if err != nil {
		return false, err
	}

	return matchesMetadataEnrichment || matchesOneAgent, nil
}

// matchOneAgent uses the namespace selector in the dynakube to check if it matches a given namespace
// if the namespace selector is not set on the dynakube its an automatic match
func matchOneAgent(dk *dynakube.DynaKube, namespace *corev1.Namespace) (bool, error) {
	if !dk.OneAgent().IsAppInjectionNeeded() {
		return false, nil
	} else if dk.OneAgent().GetNamespaceSelector() == nil {
		return true, nil
	}

	selector, err := metav1.LabelSelectorAsSelector(dk.OneAgent().GetNamespaceSelector())
	if err != nil {
		return false, errors.WithStack(err)
	}

	return selector.Matches(labels.Set(namespace.Labels)), nil
}

func matchMetadataEnrichment(dk *dynakube.DynaKube, namespace *corev1.Namespace) (bool, error) {
	if !dk.MetadataEnrichmentEnabled() {
		return false, nil
	} else if dk.MetadataEnrichmentNamespaceSelector() == nil {
		return true, nil
	}

	metadataEnrichmentSelector, err := metav1.LabelSelectorAsSelector(dk.MetadataEnrichmentNamespaceSelector())
	if err != nil {
		return false, errors.WithStack(err)
	}

	return metadataEnrichmentSelector.Matches(labels.Set(namespace.Labels)), nil
}

// updateNamespace tries to match the namespace to every dynakube with codeModules
// finds conflicting dynakubes(2 dynakube with codeModules on the same namespace)
// adds/updates/removes labels from the namespace.
func updateNamespace(namespace *corev1.Namespace, deployedDynakubes *dynakube.DynaKubeList) (bool, error) {
	namespaceUpdated := false
	conflict := ConflictChecker{}

	for i := range deployedDynakubes.Items {
		dk := &deployedDynakubes.Items[i]

		matches, err := match(dk, namespace)
		if err != nil {
			return namespaceUpdated, err
		}

		if matches {
			if err := conflict.check(dk); err != nil {
				return namespaceUpdated, err
			}
		}

		labelsUpdated := updateLabels(matches, dk, namespace)
		namespaceUpdated = labelsUpdated || namespaceUpdated
	}

	return namespaceUpdated, nil
}

func updateLabels(matches bool, dk *dynakube.DynaKube, namespace *corev1.Namespace) bool {
	updated := false

	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}

	associatedDynakubeName, instanceLabelFound := namespace.Labels[dtwebhook.InjectionInstanceLabel]

	if matches {
		if !instanceLabelFound || associatedDynakubeName != dk.Name {
			updated = true

			addNamespaceInjectLabel(dk.Name, namespace)
			log.Info("started monitoring namespace", "namespace", namespace.Name, "dk", dk.Name)
		}
	} else if instanceLabelFound && associatedDynakubeName == dk.Name {
		updated = true

		delete(namespace.Labels, dtwebhook.InjectionInstanceLabel)
	}

	return updated
}

func isIgnoredNamespace(dk *dynakube.DynaKube, namespaceName string) bool {
	for _, pattern := range dk.FF().GetIgnoredNamespaces(dk.Namespace) {
		if matched, _ := regexp.MatchString(pattern, namespaceName); matched {
			return true
		}
	}

	return false
}
