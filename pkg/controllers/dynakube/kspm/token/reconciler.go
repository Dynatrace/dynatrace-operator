package token

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
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
	dk      *dynakube.DynaKube
	secrets k8ssecret.QueryObject
}

type ReconcilerBuilder func(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler

func NewReconciler(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		dk:      dk,
		secrets: k8ssecret.Query(client, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dk.KSPM().IsEnabled() {
		return r.ensureKSPMSecret(ctx)
	}

	return r.removeKSPMSecret(ctx)
}

func (r *Reconciler) ensureKSPMSecret(ctx context.Context) error {
	_, err := r.secrets.Get(ctx, types.NamespacedName{Name: r.dk.KSPM().GetTokenSecretName(), Namespace: r.dk.Namespace})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating new token for kspm")

		secretConfig, err := generateKSPMTokenSecret(r.dk.KSPM().GetTokenSecretName(), r.dk)
		if err != nil {
			conditions.SetSecretGenFailed(r.dk.Conditions(), kspmConditionType, err)

			return err
		}

		tokenHash, err := hasher.GenerateHash(secretConfig.Data)
		if err != nil {
			conditions.SetSecretGenFailed(r.dk.Conditions(), kspmConditionType, err)

			return err
		}

		err = r.secrets.Create(ctx, secretConfig)
		if err != nil {
			log.Info("could not create secret for kspm token", "name", secretConfig.Name)
			conditions.SetKubeAPIError(r.dk.Conditions(), kspmConditionType, err)

			return err
		}

		r.dk.KSPM().TokenSecretHash = tokenHash
		conditions.SetSecretCreated(r.dk.Conditions(), kspmConditionType, r.dk.KSPM().GetTokenSecretName())
	} else if err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), kspmConditionType, err)

		return err
	}

	return nil
}

func (r *Reconciler) removeKSPMSecret(ctx context.Context) error {
	if meta.FindStatusCondition(*r.dk.Conditions(), kspmConditionType) == nil {
		return nil // no condition == nothing is there to clean up
	}

	err := r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: r.dk.KSPM().GetTokenSecretName(), Namespace: r.dk.Namespace}})
	if err != nil {
		log.Info("could not delete kspm token", "name", r.dk.KSPM().GetTokenSecretName())

		return err
	}

	r.dk.KSPM().TokenSecretHash = ""
	meta.RemoveStatusCondition(r.dk.Conditions(), kspmConditionType)

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
