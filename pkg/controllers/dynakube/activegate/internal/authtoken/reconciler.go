package authtoken

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ActiveGateAuthTokenName = "auth-token"

	// Buffer to avoid warnings in the UI
	AuthTokenBuffer           = time.Hour * 24
	AuthTokenRotationInterval = time.Hour*24*30 - AuthTokenBuffer
)

type Reconciler struct {
	secrets k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		secrets: k8ssecret.Query(clt, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, agClient agclient.APIClient, dk *dynakube.DynaKube) error {
	if !dk.ActiveGate().IsEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType) == nil {
			return nil
		}

		defer meta.RemoveStatusCondition(dk.Conditions(), ActiveGateAuthTokenSecretConditionType)

		secret, _ := k8ssecret.Build(dk, dk.ActiveGate().GetAuthTokenSecretName(), nil)
		_ = r.deleteSecret(ctx, dk, secret)

		return nil
	}

	err := r.reconcileAuthTokenSecret(ctx, dk, agClient)
	if err != nil {
		return errors.WithMessage(err, "failed to create activeGateAuthToken secret")
	}

	return nil
}

func (r *Reconciler) reconcileAuthTokenSecret(ctx context.Context, dk *dynakube.DynaKube, agClient agclient.APIClient) error {
	secret, err := r.secrets.Get(ctx, client.ObjectKey{Name: dk.ActiveGate().GetAuthTokenSecretName(), Namespace: dk.Namespace})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating activeGateAuthToken secret")

			return r.ensureAuthTokenSecret(ctx, dk, agClient)
		}

		k8sconditions.SetKubeAPIError(dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return errors.WithStack(err)
	}

	if isSecretOutdated(secret) {
		log.Info("activeGateAuthToken is outdated, creating new one")

		k8sconditions.SetSecretOutdated(dk.Conditions(), ActiveGateAuthTokenSecretConditionType, "secret is outdated, update in progress")

		if err := r.deleteSecret(ctx, dk, secret); err != nil {
			return errors.WithStack(err)
		}

		return r.ensureAuthTokenSecret(ctx, dk, agClient)
	}

	r.conditionSetSecretCreated(dk, secret) // update message once a day

	return nil
}

func (r *Reconciler) ensureAuthTokenSecret(ctx context.Context, dk *dynakube.DynaKube, agClient agclient.APIClient) error {
	agSecretData, err := r.getActiveGateAuthToken(ctx, dk, agClient)
	if err != nil {
		return errors.WithMessagef(err, "failed to create secret '%s'", dk.ActiveGate().GetAuthTokenSecretName())
	}

	return r.createSecret(ctx, dk, agSecretData)
}

func (r *Reconciler) getActiveGateAuthToken(ctx context.Context, dk *dynakube.DynaKube, agClient agclient.APIClient) (map[string][]byte, error) {
	authTokenInfo, err := agClient.GetAuthToken(ctx, dk.Name)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return nil, errors.WithStack(err)
	}

	return map[string][]byte{
		ActiveGateAuthTokenName: []byte(authTokenInfo.Token),
	}, nil
}

func (r *Reconciler) createSecret(ctx context.Context, dk *dynakube.DynaKube, secretData map[string][]byte) error {
	secretName := dk.ActiveGate().GetAuthTokenSecretName()

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.ActiveGateComponentLabel)

	secret, err := k8ssecret.Build(dk,
		secretName,
		secretData,
		k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return errors.WithStack(err)
	}

	err = r.secrets.WithOwner(dk).Create(ctx, secret)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return errors.Errorf("failed to create secret '%s': %v", secretName, err)
	}

	r.conditionSetSecretCreated(dk, secret)

	return nil
}

func (r *Reconciler) deleteSecret(ctx context.Context, dk *dynakube.DynaKube, secret *corev1.Secret) error {
	if err := r.secrets.Delete(ctx, secret); err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return err
	}

	return nil
}

func isSecretOutdated(secret *corev1.Secret) bool {
	return secret.CreationTimestamp.Add(AuthTokenRotationInterval).Before(time.Now())
}

func (r *Reconciler) conditionSetSecretCreated(dk *dynakube.DynaKube, secret *corev1.Secret) {
	lifespan := time.Since(secret.CreationTimestamp.Time)
	days := strconv.Itoa(int(lifespan.Hours() / 24))
	tokenAllParts := strings.Split(string(secret.Data[ActiveGateAuthTokenName]), ".")
	tokenPublicPart := strings.Join(tokenAllParts[:2], ".")

	setAuthSecretCreated(dk.Conditions(), ActiveGateAuthTokenSecretConditionType, "secret created "+days+" day(s) ago, token:"+tokenPublicPart)
}
