package dynakube

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConflictingNamespaceSelector(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				NamespaceSelector: metav1.LabelSelector{
					MatchLabels: dummyLabels,
				},
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
		},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: dummyLabels2,
					},
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
	t.Run(`invalid dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorConflictingNamespaceSelector},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: dummyLabels,
					},
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
					},
				},
			},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflicting-dk",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: dummyLabels,
					},
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
					},
				},
			}, &defaultCSIDaemonSet, &dummyNamespace)
	})
	t.Run("validate namespaceSelector to be a valid label according to spec", func(t *testing.T) {
		testsValidLabels := []string{
			"",
			"a",
			"short",
			"WithUpperCase",
			"contains123",
			"label-with-Dash",
			"label_with_underscore",
			"label.with.dotttses",
			"label.with.dotttses-567567",
		}
		// MatchLabels
		for _, label := range testsValidLabels {
			assertAllowedResponse(t, &dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-namespace-selector",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"dummy": label,
						},
					},
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
		}
		// MatchExpressions
		assertAllowedResponse(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalid-namespace-selector",
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				NamespaceSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "dummy",
							Operator: metav1.LabelSelectorOpIn,
							Values:   testsValidLabels,
						},
					},
				},
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
		}, &dummyNamespace, &dummyNamespace2)

		testsInvalidLabels := []string{
			"name%",
			"name/",
			"AMuchTooLongLabelThatGoesOverSixtyThreeCharactersAndSoIsInvalidAccordingToSpec",
		}
		for _, label := range testsInvalidLabels {
			// MatchLabels
			assertDeniedResponse(t,
				[]string{errorNamespaceSelectorMatchLabelsViolateLabelSpec},
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-namespace-selector",
						Namespace: testNamespace,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						APIURL: testApiUrl,
						NamespaceSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"dummy": label,
							},
						},
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
						},
					},
				}, &dummyNamespace, &dummyNamespace2)
		}
		// MatchExpressions
		assertDeniedResponse(t,
			[]string{errorNamespaceSelectorMatchLabelsViolateLabelSpec},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-namespace-selector",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "dummy",
								Operator: metav1.LabelSelectorOpIn,
								Values:   testsInvalidLabels,
							},
						},
					},
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
}
