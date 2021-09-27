package mapper

import (
	dynatracev1 "github.com/Dynatrace/dynatrace-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestDynakubeWithAppInject(name string, labels map[string]string, labelExpression []metav1.LabelSelectorRequirement) *dynatracev1.DynaKube {
	dk := &dynatracev1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dynatrace"},
		Spec: dynatracev1.DynaKubeSpec{
			OneAgent: dynatracev1.OneAgentSpec{
				ApplicationMonitoring: &dynatracev1.ApplicationMonitoringSpec{},
			},
		},
	}
	if labels != nil {
		dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = metav1.LabelSelector{MatchLabels: labels}
	}
	if labelExpression != nil {
		dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = metav1.LabelSelector{MatchExpressions: labelExpression}
	}
	return dk
}

func createTestDynakubeWithMultipleFeatures(name string, labels map[string]string, labelExpression []metav1.LabelSelectorRequirement) *dynatracev1.DynaKube {
	dk := createTestDynakubeWithAppInject(name, labels, labelExpression)
	dk.Spec.Routing.Enabled = true
	return dk
}

func createNamespace(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}
