package dynakube

import ctrl "sigs.k8s.io/controller-runtime"

func (r *DynaKube) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}
