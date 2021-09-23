package mapper

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestBlankDynakube(name string, labels map[string]string, labelExpression []metav1.LabelSelectorRequirement) *dynatracev1alpha1.DynaKube {
	dk := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dynatrace"},
	}
	if labels != nil {
		dk.Spec.NamespaceSelector = &metav1.LabelSelector{MatchLabels: labels}
	}
	if labelExpression != nil {
		dk.Spec.NamespaceSelector = &metav1.LabelSelector{MatchExpressions: labelExpression}
	}
	return dk
}

func createTestDynakubeWithMultipleFeatures(name string, labels map[string]string, labelExpression []metav1.LabelSelectorRequirement) *dynatracev1alpha1.DynaKube {
	dk := createTestBlankDynakube(name, labels, labelExpression)
	dk.Spec.CodeModules.Enabled = true
	dk.Spec.DataIngestSpec.Enabled = true
	return dk
}

func createTestDynakubeWithCodeModules(name string, labels map[string]string, labelExpression []metav1.LabelSelectorRequirement) *dynatracev1alpha1.DynaKube {
	dk := createTestBlankDynakube(name, labels, labelExpression)
	dk.Spec.CodeModules.Enabled = true
	return dk
}

func createTestDynakubeWithDataIngest(name string, labels map[string]string, labelExpression []metav1.LabelSelectorRequirement) *dynatracev1alpha1.DynaKube {
	dk := createTestBlankDynakube(name, labels, labelExpression)
	dk.Spec.DataIngestSpec.Enabled = true
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
