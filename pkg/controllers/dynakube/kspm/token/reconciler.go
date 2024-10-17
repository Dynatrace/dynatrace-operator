package token

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

type ReconcilerBuilder func(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler

func NewReconciler(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		client:    client,
		dk:        dk,
		apiReader: apiReader,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dk.IsKSPMEnabled() {
		return ensureKSPMSecret(ctx, r.client, r.apiReader, r.dk)
	}

	return removeKSPMSecret(ctx, r.client, r.apiReader, r.dk)
}

func ensureKSPMSecret(ctx context.Context, client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) error {
	query := k8ssecret.Query(client, apiReader, log)
	_, err := query.Get(ctx, types.NamespacedName{Name: dk.GetKSPMSecretName(), Namespace: dk.Namespace})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating new token for kspm", "error", err.Error())

		secretConfig, err := generateKSPMTokenSecret(dk.GetKSPMSecretName(), dk)

		if err != nil {
			return err
		}

		err = query.Create(ctx, secretConfig)
		if err != nil {
			log.Info("could not create secret for kspm token", "name", secretConfig.Name)

			return err
		}
	}

	return nil
}

func removeKSPMSecret(ctx context.Context, client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) error {
	query := k8ssecret.Query(client, apiReader, log)
	err := query.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.GetKSPMSecretName(), Namespace: dk.Namespace}})

	if err != nil {
		log.Info("could not delete kspm token", "name", dk.GetKSPMSecretName())

		return err
	}

	return nil
}

func generateKSPMTokenSecret(name string, dk *dynakube.DynaKube) (secret *corev1.Secret, err error) {
	newToken, err := dttoken.New("dt0n01")
	if err != nil {
		return nil, err
	}

	secretData := make(map[string][]byte)
	secretData[dynakube.KSPMSecretKey] = []byte(newToken.String())
	secretConfig, err := k8ssecret.Build(dk,
		name,
		secretData,
	)

	if err != nil {
		return nil, err
	}

	return secretConfig, nil
}
