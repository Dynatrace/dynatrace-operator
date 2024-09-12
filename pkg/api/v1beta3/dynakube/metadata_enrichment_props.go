package dynakube

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (dk *DynaKube) MetadataEnrichmentEnabled() bool {
	return dk.Spec.MetadataEnrichment.Enabled
}

func (dk *DynaKube) MetadataEnrichmentNamespaceSelector() *metav1.LabelSelector {
	return &dk.Spec.MetadataEnrichment.NamespaceSelector
}
