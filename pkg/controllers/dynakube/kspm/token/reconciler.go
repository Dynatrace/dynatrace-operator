package token

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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
	if r.dk.KSPM().IsEnabled() {
		return ensureKSPMSecret(ctx, r.client, r.apiReader, r.dk)
	}

	return removeKSPMSecret(ctx, r.client, r.apiReader, r.dk)
}

func ensureKSPMSecret(ctx context.Context, client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) error {
	query := k8ssecret.Query(client, apiReader, log)
	_, err := query.Get(ctx, types.NamespacedName{Name: dk.KSPM().GetTokenSecretName(), Namespace: dk.Namespace})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating new token for kspm")

		secretConfig, err := generateKSPMTokenSecret(dk.KSPM().GetTokenSecretName(), dk)

		if err != nil {
			conditions.SetSecretGenFailed(dk.Conditions(), kspmConditionType, err)

			return err
		}

		err = query.Create(ctx, secretConfig)
		if err != nil {
			log.Info("could not create secret for kspm token", "name", secretConfig.Name)
			conditions.SetKubeApiError(dk.Conditions(), kspmConditionType, err)

			return err
		}

		conditions.SetSecretCreated(dk.Conditions(), kspmConditionType, dk.KSPM().GetTokenSecretName())
	}

	return nil
}

func removeKSPMSecret(ctx context.Context, client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) error {
	if meta.FindStatusCondition(*dk.Conditions(), kspmConditionType) == nil {
		return nil // no condition == nothing is there to clean up
	}

	query := k8ssecret.Query(client, apiReader, log)
	err := query.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.KSPM().GetTokenSecretName(), Namespace: dk.Namespace}})

	if err != nil {
		log.Info("could not delete kspm token", "name", dk.KSPM().GetTokenSecretName())

		return err
	}

	meta.RemoveStatusCondition(dk.Conditions(), kspmConditionType)

	return nil
}

func generateKSPMTokenSecret(name string, dk *dynakube.DynaKube) (secret *corev1.Secret, err error) {
	newToken, err := dttoken.New("dt0n01")
	if err != nil {
		return nil, err
	}

	secretData := make(map[string][]byte)
	secretData[kspm.TokenSecretKey] = []byte(newToken.String())
	secretConfig, err := k8ssecret.Build(dk,
		name,
		secretData,
	)

	if err != nil {
		return nil, err
	}

	return secretConfig, nil
}
