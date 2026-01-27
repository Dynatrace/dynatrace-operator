package validation

import (
	v1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	v1alpha2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validator := New(mgr.GetAPIReader(), mgr.GetConfig())

	if err := ctrl.NewWebhookManagedBy(mgr, &v1alpha1.EdgeConnect{}).
		WithCustomValidator(validator). //nolint
		Complete(); err != nil {
		return err
	}

	return ctrl.NewWebhookManagedBy(mgr, &v1alpha2.EdgeConnect{}).
		WithCustomValidator(validator). //nolint
		Complete()
}
