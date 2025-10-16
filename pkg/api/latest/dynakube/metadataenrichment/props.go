package metadataenrichment

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (enrichment *MetadataEnrichment) GetNamespaceSelector() *metav1.LabelSelector {
	return &enrichment.NamespaceSelector
}
