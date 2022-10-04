package secret

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	secret    *corev1.Secret
}

var _ kubeobjects.Reconciler = &Reconciler{}

const (
	CommunicationEndpointsName = "communication-endpoints"
	TenantTokenName            = "tenant-token"
	TenantUuidName             = "tenant-uuid"
)

func NewReconciler(clt client.Client, apiReader client.Reader, secret *corev1.Secret) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		secret:    secret,
	}
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	query := kubeobjects.NewSecretQuery(context.TODO(), r.client, r.apiReader, log)
	return true, query.CreateOrUpdate(*r.secret)
}
