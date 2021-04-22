package dtlabels

import (
	"strings"

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
		requirement, err := labels.NewRequirement(expression.Key, requirementOperatorToSelectionOperator(expression.Operator), expression.Values)
		if err != nil {
			return false, errors.WithStack(err)
		}
		selector = selector.Add(*requirement)
	}
	return selector.Matches(labels.Set(objectLabels)), nil
}

func requirementOperatorToSelectionOperator(labelSelectionOperator metav1.LabelSelectorOperator) selection.Operator {
	// Label Selector Operators are Capitalized (e.g. 'In'), while Operators from the selection package are lowercase (i.e. 'in')
	// Both are strings in the background and need some conversion to work together
	return selection.Operator(strings.ToLower(string(labelSelectionOperator)))
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
