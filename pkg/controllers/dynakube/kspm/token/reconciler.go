package token

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	secrets k8ssecret.QueryObject
}

func NewReconciler(client client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		secrets: k8ssecret.Query(client, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	if dk.KSPM().IsEnabled() {
		return r.ensureKSPMSecret(ctx, dk)
	}

	return r.removeKSPMSecret(ctx, dk)
}

func (r *Reconciler) ensureKSPMSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	_, err := r.secrets.Get(ctx, types.NamespacedName{Name: dk.KSPM().GetTokenSecretName(), Namespace: dk.Namespace})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating new token for kspm")

		secretConfig, err := generateKSPMTokenSecret(dk.KSPM().GetTokenSecretName(), dk)
		if err != nil {
			k8sconditions.SetSecretGenFailed(dk.Conditions(), kspmConditionType, err)

			return err
		}

		tokenHash, err := hasher.GenerateHash(secretConfig.Data)
		if err != nil {
			k8sconditions.SetSecretGenFailed(dk.Conditions(), kspmConditionType, err)

			return err
		}

		err = r.secrets.Create(ctx, secretConfig)
		if err != nil {
			log.Info("could not create secret for kspm token", "name", secretConfig.Name)
			k8sconditions.SetKubeAPIError(dk.Conditions(), kspmConditionType, err)

			return err
		}

		dk.KSPM().TokenSecretHash = tokenHash
		k8sconditions.SetSecretCreated(dk.Conditions(), kspmConditionType, dk.KSPM().GetTokenSecretName())
	} else if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), kspmConditionType, err)

		return err
	}

	return nil
}

func (r *Reconciler) removeKSPMSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	if meta.FindStatusCondition(*dk.Conditions(), kspmConditionType) == nil {
		return nil // no condition == nothing is there to clean up
	}

	err := r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.KSPM().GetTokenSecretName(), Namespace: dk.Namespace}})
	if err != nil {
		log.Info("could not delete kspm token", "name", dk.KSPM().GetTokenSecretName())

		return err
	}

	dk.KSPM().TokenSecretHash = ""
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
		k8ssecret.SetLabels(k8slabel.NewCoreLabels(dk.Name, k8slabel.KSPMComponentLabel).BuildLabels()),
	)
	if err != nil {
		return nil, err
	}

	return secretConfig, nil
}
