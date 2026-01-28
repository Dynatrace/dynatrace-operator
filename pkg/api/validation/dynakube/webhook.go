package validation

import (
	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	v1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	v1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	v1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validator := New(mgr.GetAPIReader(), mgr.GetConfig())

	if err := ctrl.NewWebhookManagedBy(mgr, &v1beta3.DynaKube{}).
		WithCustomValidator(validator).
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewWebhookManagedBy(mgr, &v1beta4.DynaKube{}).
		WithCustomValidator(validator).
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewWebhookManagedBy(mgr, &v1beta5.DynaKube{}).
		WithCustomValidator(validator).
		Complete(); err != nil {
		return err
	}

	return ctrl.NewWebhookManagedBy(mgr, &latest.DynaKube{}).
		WithCustomValidator(validator).
		Complete()
}
