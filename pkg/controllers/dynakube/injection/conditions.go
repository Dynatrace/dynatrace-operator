package injection

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	metaDataEnrichmentConditionType        = "MetadataEnrichment"
	codeModulesInjectionConditionType      = "CodeModulesInjection"
	otlpExporterConfigurationConditionType = "OTLPExporterConfiguration"

	secretsCreatedReason  = "SecretsCreated"
	secretsCreatedMessage = "Namespaces mapped and secrets created"
)

func setMetadataEnrichmentCreatedCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    metaDataEnrichmentConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  secretsCreatedReason,
		Message: secretsCreatedMessage,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setCodeModulesInjectionCreatedCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    codeModulesInjectionConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  secretsCreatedReason,
		Message: secretsCreatedMessage,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setOTLPExporterConfigurationCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    otlpExporterConfigurationConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  secretsCreatedReason,
		Message: secretsCreatedMessage,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
