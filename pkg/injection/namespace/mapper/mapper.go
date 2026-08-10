// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package mapper

import (
	"context"
	"regexp"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ErrorConflictingNamespace = "namespace matches two or more DynaKubes which is unsupported. " +
		"refine the labels on your namespace metadata or DynaKube/CodeModules specification"
)

type ConflictChecker struct {
	alreadyUsed bool
}

func (c *ConflictChecker) check(dk *dynakube.DynaKube) error {
	metadataEnrichment := dk.MetadataEnrichment()
	otlpExporterConfig := dk.OTLPExporterConfiguration()

	if !dk.OneAgent().IsAppInjectionNeeded() &&
		!metadataEnrichment.IsEnabled() &&
		!otlpExporterConfig.IsEnabled() {
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

type compiledSelectors struct {
	oneAgent labels.Selector
	metadata labels.Selector
	otlp     labels.Selector
}

// compileSelectors compiles the DynaKube's namespace selectors. If any selector is invalid it will be nil.
// This function assumes that invalid selectors would've been rejected by the validation webhook.
func compileSelectors(dk *dynakube.DynaKube) compiledSelectors {
	var selectors compiledSelectors

	if dk.OneAgent().IsAppInjectionNeeded() {
		selectors.oneAgent, _ = metav1.LabelSelectorAsSelector(dk.OneAgent().GetNamespaceSelector())
	}

	if metadataEnrichment := dk.MetadataEnrichment(); metadataEnrichment.IsEnabled() {
		selectors.metadata, _ = metav1.LabelSelectorAsSelector(&metadataEnrichment.NamespaceSelector)
	}

	if otlpExporterConfiguration := dk.OTLPExporterConfiguration(); otlpExporterConfiguration.IsEnabled() {
		selectors.otlp, _ = metav1.LabelSelectorAsSelector(&otlpExporterConfiguration.Spec.NamespaceSelector)
	}

	return selectors
}

func match(dk *dynakube.DynaKube, namespace *corev1.Namespace, selectors compiledSelectors) matchFlags {
	var flags matchFlags

	if isIgnoredNamespace(dk, namespace.Name) {
		return flags
	}

	set := labels.Set(namespace.Labels)

	if selectors.oneAgent != nil && selectors.oneAgent.Matches(set) {
		flags |= flagOneAgent
	}

	if selectors.metadata != nil && selectors.metadata.Matches(set) {
		flags |= flagMetadata
	}

	if selectors.otlp != nil && selectors.otlp.Matches(set) {
		flags |= flagOTLP
	}

	return flags
}

// updateNamespace tries to match the namespace to every dynakube with codeModules
// finds conflicting dynakubes(2 dynakube with codeModules on the same namespace)
// adds/updates/removes labels from the namespace.
func updateNamespace(ctx context.Context, namespace *corev1.Namespace, deployedDynakubes *dynakube.DynaKubeList) (bool, error) {
	namespaceUpdated := false
	conflict := ConflictChecker{}

	for i := range deployedDynakubes.Items {
		dk := &deployedDynakubes.Items[i]

		// Recompiled per namespace this function is called for, since this loop runs once per
		// namespace (from mapFromDynakube's caller): O(namespaces x dynakubes) instead of O(dynakubes).
		// Left as-is - dynakubes with a relevant selector are rarely more than one or two.
		selectors := compileSelectors(dk)
		flags := match(dk, namespace, selectors)

		if flags.isAny() {
			if err := conflict.check(dk); err != nil {
				return namespaceUpdated, err
			}
		}

		labelsUpdated := updateLabels(ctx, dk, namespace, flags.isAny())
		namespaceUpdated = labelsUpdated || namespaceUpdated
	}

	return namespaceUpdated, nil
}

func updateLabels(ctx context.Context, dk *dynakube.DynaKube, namespace *corev1.Namespace, matches bool) bool {
	associatedDynakubeName, instanceLabelFound := namespace.Labels[dtwebhook.InjectionInstanceLabel]
	updated := false

	if matches {
		if !instanceLabelFound || associatedDynakubeName != dk.Name {
			updated = true

			addNamespaceInjectLabel(dk.Name, namespace)
			logd.FromContext(ctx).Info("started monitoring namespace", "namespace", namespace.Name)
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
