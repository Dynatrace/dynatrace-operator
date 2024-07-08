package extension

import (
	"context"

	dynatracev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client       client.Client
	apiReader    client.Reader
	timeProvider *timeprovider.Provider

	dynakube *dynatracev1beta3.DynaKube
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta3.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

const (
	tokenKey         = "eec-token"
	tokenValuePrefix = "EEC dt0x01"
	secretSuffix     = "-extensions-token"
)

func NewReconciler(clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta3.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:       clt,
		apiReader:    apiReader,
		dynakube:     dynakube,
		timeProvider: timeprovider.New(),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	log.Info("start reconciling extensions")

	if r.dynakube.PrometheusEnabled() {
		err := r.reconcileSecret(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) reconcileSecret(ctx context.Context) error {
	log.Info("reconciling secret")

	query := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)

	_, err := query.Get(client.ObjectKey{Name: r.secretName(), Namespace: r.dynakube.Namespace})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if k8serrors.IsNotFound(err) {
		log.Info("creating secret")

		newToken, err := dttoken.New(tokenValuePrefix)
		if err != nil {
			return err
		}

		newSecret, err := r.buildSecret(*newToken)
		if err != nil {
			return err
		}

		query.CreateOrUpdate(*newSecret)
	}

	return nil
}

func (r *reconciler) secretName() string {
	return r.dynakube.Name + secretSuffix
}

func (r *reconciler) buildSecret(token dttoken.Token) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		tokenKey: []byte(token.String()),
	}

	return k8ssecret.Create(r.dynakube, k8ssecret.NewNameModifier(r.secretName()), k8ssecret.NewNamespaceModifier(r.dynakube.GetNamespace()), k8ssecret.NewDataModifier(secretData))
}
