package dtlabels

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func IsMatching(matchLabels map[string]string, matchExpressions []metav1.LabelSelectorRequirement, objectLabels map[string]string) (bool, error) {
	if AreLabelsMatching(matchLabels, objectLabels) {
		matching, err := AreExpressionsMatching(matchExpressions, objectLabels)
		if err != nil {
			return false, errors.WithStack(err)
		}
		return matching, nil
	}
	return false, nil
}

func AreExpressionsMatching(expressions []metav1.LabelSelectorRequirement, objectLabels map[string]string) (bool, error) {
	selector := labels.NewSelector()
	for _, expression := range expressions {
		requirement, err := labels.NewRequirement(
			expression.Key,
			requirementOperatorToSelectionOperator(expression.Operator),
			expression.Values)
		if err != nil {
			return false, errors.WithStack(err)
		}
		selector = selector.Add(*requirement)
	}
	return selector.Matches(labels.Set(objectLabels)), nil
}

func requirementOperatorToSelectionOperator(labelSelectionOperator metav1.LabelSelectorOperator) selection.Operator {
	// LabelSelectorOperator for LabelSelectorRequirements differ from the operators used by the selection
	// package which is used to match labels against the requirements. Therefore they need some mapping.
	// The switch below maps the existing four LabelSelectorOperators to their selection.Operator counterpart
	switch labelSelectionOperator {
	case metav1.LabelSelectorOpIn:
		return selection.In
	case metav1.LabelSelectorOpNotIn:
		return selection.NotIn
	case metav1.LabelSelectorOpExists:
		return selection.Exists
	case metav1.LabelSelectorOpDoesNotExist:
		return selection.DoesNotExist
	}

	// Returning an invalid operator here results in an error when a new requirement is instantiated.
	// This error is then propagated correctly.
	// Therefore no error is returned here so this function can be inlined.
	return ""
}

func AreLabelsMatching(matchLabels map[string]string, labels map[string]string) bool {
	if len(labels) == 0 {
		return false
	}

	for matchLabel, matchValue := range matchLabels {
		value, ok := labels[matchLabel]
		if !ok || matchValue != value {
			return false
		}
	}
	return true
}
