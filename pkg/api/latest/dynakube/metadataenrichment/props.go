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

func (m *MetadataEnrichment) IsEnabled() bool {
	return m.Enabled != nil && *m.Enabled
}

func (m *MetadataEnrichment) GetNamespaceSelector() *v1.LabelSelector {
	// return &dk.Spec.MetadataEnrichment.NamespaceSelector
	return &m.NamespaceSelector
}
