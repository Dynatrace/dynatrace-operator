package edgeconnect

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func SetupWebhookWithManager(mgr ctrl.Manager, validator admission.CustomValidator) error {
	return ctrl.NewWebhookManagedBy(mgr, &EdgeConnect{}).
		WithCustomValidator(validator). // will create an endpoint at /validate-dynatrace-com-v1alpha2-edgeconnect
		Complete()
}
