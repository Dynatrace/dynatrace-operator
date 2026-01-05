package validation

import (
	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	v1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	v1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validator := New(mgr.GetAPIReader(), mgr.GetConfig())

	if err := v1beta4.SetupWebhookWithManager(mgr, validator); err != nil {
		return err
	}

	if err := v1beta5.SetupWebhookWithManager(mgr, validator); err != nil {
		return err
	}

	return latest.SetupWebhookWithManager(mgr, validator)
}
