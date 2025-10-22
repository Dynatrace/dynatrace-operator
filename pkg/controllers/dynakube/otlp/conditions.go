package otlp

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	otlpExporterConfigurationConditionType = "OTLPExporterConfiguration"

	secretsCreatedReason  = "SecretsCreated"
	secretsCreatedMessage = "Namespaces mapped and secrets created"
)

func setOTLPExporterConfigurationCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    otlpExporterConfigurationConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  secretsCreatedReason,
		Message: secretsCreatedMessage,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
