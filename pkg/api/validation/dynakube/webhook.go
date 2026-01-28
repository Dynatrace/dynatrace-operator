package validation

import (
	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	v1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	v1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	v1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func setupWebhookWithManager(mgr ctrl.Manager, obj runtime.Object, validator admission.Validator[runtime.Object]) error {
	return ctrl.NewWebhookManagedBy(mgr, obj).WithCustomValidator(validator).Complete() //nolint
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validator := New(mgr.GetAPIReader(), mgr.GetConfig())

	if err := setupWebhookWithManager(mgr, &v1beta3.DynaKube{}, validator); err != nil {
		return err
	}

	if err := setupWebhookWithManager(mgr, &v1beta4.DynaKube{}, validator); err != nil {
		return err
	}

	if err := setupWebhookWithManager(mgr, &v1beta5.DynaKube{}, validator); err != nil {
		return err
	}

	return setupWebhookWithManager(mgr, &latest.DynaKube{}, validator)
}
