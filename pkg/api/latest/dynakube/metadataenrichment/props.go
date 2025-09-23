package metadataenrichment

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (rule EnrichmentRule) ToAnnotationKey() string {
	if rule.Target == "" {
		return ""
	}

	return Prefix + rule.Target
}

func GetEmptyTargetEnrichmentKey(metadataType, key string) string {
	return namespaceKeyPrefix + strings.ToLower(metadataType) + "." + key
}

func (enrichment *MetadataEnrichment) IsEnabled() bool {
	return enrichment.Enabled != nil && *enrichment.Enabled
}

func (enrichment *MetadataEnrichment) GetNamespaceSelector() *v1.LabelSelector {
	// return &dk.Spec.MetadataEnrichment.NamespaceSelector
	return &enrichment.NamespaceSelector
}
