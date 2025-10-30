package metadataenrichment

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r Rule) ToAnnotationKey() string {
	if r.Target == "" {
		return ""
	}

	return Prefix + r.Target
}

func GetEmptyTargetEnrichmentKey(metadataType, key string) string {
	return namespaceKeyPrefix + strings.ToLower(metadataType) + "." + key
}

func (m *MetadataEnrichment) IsEnabled() bool {
	return m.Enabled != nil && *m.Enabled
}

func (m *MetadataEnrichment) GetNamespaceSelector() *metav1.LabelSelector {
	return &m.NamespaceSelector
}
