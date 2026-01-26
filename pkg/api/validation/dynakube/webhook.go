package validation

import (
	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	v1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	v1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	v1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validatorImpl := newClient(mgr.GetAPIReader(), mgr.GetConfig())

	v1beta3Validator := newValidator[*v1beta3.DynaKube](validatorImpl)
	if err := ctrl.NewWebhookManagedBy(mgr, &v1beta3.DynaKube{}).
		WithValidator(v1beta3Validator). // will create an endpoint at /validate-dynatrace-com-v1beta3-dynakube
		Complete(); err != nil {
		return err
	}

	v1beta4Validator := newValidator[*v1beta4.DynaKube](validatorImpl)
	if err := ctrl.NewWebhookManagedBy(mgr, &v1beta4.DynaKube{}).
		WithValidator(v1beta4Validator). // will create an endpoint at /validate-dynatrace-com-v1beta4-dynakube
		Complete(); err != nil {
		return err
	}

	v1beta5Validator := newValidator[*v1beta5.DynaKube](validatorImpl)
	if err := ctrl.NewWebhookManagedBy(mgr, &v1beta5.DynaKube{}).
		WithValidator(v1beta5Validator). // will create an endpoint at /validate-dynatrace-com-v1beta4-dynakube
		Complete(); err != nil {
		return err
	}

	latestValidator := newValidator[*latest.DynaKube](validatorImpl)
	if err := ctrl.NewWebhookManagedBy(mgr, &latest.DynaKube{}).
		WithValidator(latestValidator). // will create an endpoint at /validate-dynatrace-com-v1beta4-dynakube
		Complete(); err != nil {
		return err
	}

	return nil
}
