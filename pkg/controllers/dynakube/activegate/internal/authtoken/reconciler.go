package authtoken

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
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

var _ controllers.Reconciler = &Reconciler{}

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
	dtc       dtclient.Client
}

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube, dtc dtclient.Client) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
		dtc:       dtc,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.ActiveGate().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), ActiveGateAuthTokenSecretConditionType) == nil {
			return nil
		}

		defer meta.RemoveStatusCondition(r.dk.Conditions(), ActiveGateAuthTokenSecretConditionType)

		secret, _ := k8ssecret.Build(r.dk, r.dk.ActiveGate().GetAuthTokenSecretName(), nil)
		_ = r.deleteSecret(ctx, secret)

		return nil
	}

	err := r.reconcileAuthTokenSecret(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to create activeGateAuthToken secret")
	}

	return nil
}

func (r *Reconciler) reconcileAuthTokenSecret(ctx context.Context) error {
	secret, err := r.secretQuery().Get(ctx, client.ObjectKey{Name: r.dk.ActiveGate().GetAuthTokenSecretName(), Namespace: r.dk.Namespace})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating activeGateAuthToken secret")

			return r.ensureAuthTokenSecret(ctx)
		}

		conditions.SetKubeApiError(r.dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return errors.WithStack(err)
	}

	if isSecretOutdated(secret) {
		log.Info("activeGateAuthToken is outdated, creating new one")

		conditions.SetSecretOutdated(r.dk.Conditions(), ActiveGateAuthTokenSecretConditionType, "secret is outdated, update in progress")

		if err := r.deleteSecret(ctx, secret); err != nil {
			return errors.WithStack(err)
		}

		return r.ensureAuthTokenSecret(ctx)
	}

	r.conditionSetSecretCreated(secret) // update message once a day

	return nil
}

func (r *Reconciler) ensureAuthTokenSecret(ctx context.Context) error {
	agSecretData, err := r.getActiveGateAuthToken(ctx)
	if err != nil {
		return errors.WithMessagef(err, "failed to create secret '%s'", r.dk.ActiveGate().GetAuthTokenSecretName())
	}

	return r.createSecret(ctx, agSecretData)
}

func (r *Reconciler) getActiveGateAuthToken(ctx context.Context) (map[string][]byte, error) {
	authTokenInfo, err := r.dtc.GetActiveGateAuthToken(ctx, r.dk.Name)
	if err != nil {
		conditions.SetDynatraceApiError(r.dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return nil, errors.WithStack(err)
	}

	return map[string][]byte{
		ActiveGateAuthTokenName: []byte(authTokenInfo.Token),
	}, nil
}

func (r *Reconciler) createSecret(ctx context.Context, secretData map[string][]byte) error {
	secretName := r.dk.ActiveGate().GetAuthTokenSecretName()

	secret, err := k8ssecret.Build(r.dk,
		secretName,
		secretData)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return errors.WithStack(err)
	}

	err = r.secretQuery().WithOwner(r.dk).Create(ctx, secret)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return errors.Errorf("failed to create secret '%s': %v", secretName, err)
	}

	r.conditionSetSecretCreated(secret)

	return nil
}

func (r *Reconciler) deleteSecret(ctx context.Context, secret *corev1.Secret) error {
	if err := r.secretQuery().Delete(ctx, secret); err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), ActiveGateAuthTokenSecretConditionType, err)

		return err
	}

	return nil
}

func isSecretOutdated(secret *corev1.Secret) bool {
	return secret.CreationTimestamp.Add(AuthTokenRotationInterval).Before(time.Now())
}

func (r *Reconciler) conditionSetSecretCreated(secret *corev1.Secret) {
	lifespan := time.Since(secret.CreationTimestamp.Time)
	days := strconv.Itoa(int(lifespan.Hours() / 24))
	tokenAllParts := strings.Split(string(secret.Data[ActiveGateAuthTokenName]), ".")
	tokenPublicPart := strings.Join(tokenAllParts[:2], ".")

	setAuthSecretCreated(r.dk.Conditions(), ActiveGateAuthTokenSecretConditionType, "secret created "+days+" day(s) ago, token:"+tokenPublicPart)
}

func (r *Reconciler) secretQuery() k8ssecret.QueryObject {
	return k8ssecret.Query(r.client, r.apiReader, log)
}
