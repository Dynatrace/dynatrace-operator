package validation

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func filterForValidationDeployment(namespace string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		return hasValidationWebhookName(object) && isInNamespace(object, namespace)
	})
}

func isInNamespace(object client.Object, namespace string) bool {
	return object.GetNamespace() == namespace
}

func hasValidationWebhookName(object client.Object) bool {
	return object.GetName() == validationWebhookName
}
