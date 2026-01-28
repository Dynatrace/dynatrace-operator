package validation

import (
	v1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	v1alpha2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func setupWebhookWithManager(mgr ctrl.Manager, obj runtime.Object, validator admission.Validator[runtime.Object]) error {
	return ctrl.NewWebhookManagedBy(mgr, obj).WithCustomValidator(validator).Complete() //nolint
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validator := New(mgr.GetAPIReader(), mgr.GetConfig())

	if err := setupWebhookWithManager(mgr, &v1alpha1.EdgeConnect{}, validator); err != nil {
		return err
	}

	return setupWebhookWithManager(mgr, &v1alpha2.EdgeConnect{}, validator)
}
