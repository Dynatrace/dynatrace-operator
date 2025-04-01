package mapper

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MissingError struct {
	namespace string
}

func (err MissingError) Error() string {
	return fmt.Sprintf("no dynakube matches the namespace(%s). refine the labels on your namespace metadata or DynaKube/CodeModules specification", err.namespace)
}

func GetDynakubeForNamespace(ctx context.Context, clt client.Reader, ns corev1.Namespace, dtNs string) (*dynakube.DynaKube, error) {
	if IsIgnoredNamespace(ns.Name) {
		return nil, IgnoredError{namespace: ns.Name}
	}

	dks := &dynakube.DynaKubeList{}

	err := clt.List(ctx, dks, client.InNamespace(dtNs))
	if err != nil {
		return nil, err
	}

	for _, dk := range dks.Items {
		oaMatch, err := matchOneAgent(dk, ns)
		if err != nil {
			return nil, err
		}

		meMatch, err := matchMetadataEnrichment(dk, ns)
		if err != nil {
			return nil, err
		}

		if oaMatch || meMatch {
			return &dk, nil

		}
	}

	return nil, MissingError{namespace: ns.Name}
}

func HasConflict(ctx context.Context, clt client.Reader, dk *dynakube.DynaKube) (bool, error) {
	namespaces, err := GetNamespacesForDynakube(ctx, clt, dk)
	if err != nil {
		return false, err
	}

	for _, ns := range namespaces {
		_, err := GetDynakubeForNamespace(ctx, clt, ns, dk.Namespace)
		if !errors.As(err, &MissingError{}) && !errors.As(err, &IgnoredError{}){
			return true, nil
		}
	}

	return false, nil

}

func GetNamespacesForDynakube(ctx context.Context, clt client.Reader, dk *dynakube.DynaKube) ([]corev1.Namespace, error) {
	oaList := &corev1.NamespaceList{}
	if dk.OneAgent().IsAppInjectionNeeded() {
		selector, _ := metav1.LabelSelectorAsSelector(dk.OneAgent().GetNamespaceSelector())
		listOps := []client.ListOption{
			client.MatchingLabelsSelector{Selector: selector},
		}

		err := clt.List(ctx, oaList, listOps...)
		if err != nil {
			return nil, err
		}
	}

	meList := &corev1.NamespaceList{}
	if dk.MetadataEnrichmentEnabled() {
		selector, _ := metav1.LabelSelectorAsSelector(dk.MetadataEnrichmentNamespaceSelector())
		listOps := []client.ListOption{
			client.MatchingLabelsSelector{Selector: selector},
		}

		err := clt.List(ctx, meList, listOps...)
		if err != nil {
			return nil, err
		}
	}

	allNs := map[string]corev1.Namespace{}
	for _, item := range oaList.Items {
		allNs[item.Name] = item
	}
	for _, item := range meList.Items {
		allNs[item.Name] = item
	}

	nsList := []corev1.Namespace{}
	for _, ns := range allNs {
		if IsIgnoredNamespace(ns.Name) {
			continue
		}
		nsList = append(nsList, ns)
	}

	return nsList, nil
}

// matchOneAgent uses the namespace selector in the dynakube to check if it matches a given namespace
// if the namespace selector is not set on the dynakube its an automatic match
func matchOneAgent(dk dynakube.DynaKube, namespace corev1.Namespace) (bool, error) {
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

func matchMetadataEnrichment(dk dynakube.DynaKube, namespace corev1.Namespace) (bool, error) {
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
