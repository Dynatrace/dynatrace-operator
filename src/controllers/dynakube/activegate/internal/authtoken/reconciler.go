package authtoken

import (
	"context"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	dynakube  *dynatracev1beta1.DynaKube
	scheme    *runtime.Scheme
	dtc       dtclient.Client
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		dynakube:  dynakube,
		dtc:       dtc,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.reconcileAuthTokenSecret(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to create activeGateAuthToken secret")
	}
	return nil
}

func (r *Reconciler) reconcileAuthTokenSecret(ctx context.Context) error {
	var secret corev1.Secret
	err := r.apiReader.Get(ctx,
		client.ObjectKey{Name: r.dynakube.ActiveGateAuthTokenSecret(), Namespace: r.dynakube.Namespace},
		&secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating activeGateAuthToken secret")
			return r.ensureAuthTokenSecret(ctx)
		}
		return errors.WithStack(err)
	}
	if isSecretOutdated(&secret) {
		log.Info("activeGateAuthToken is outdated, creating new one")
		if err := r.deleteSecret(ctx, &secret); err != nil {
			return errors.WithStack(err)
		}
		return r.ensureAuthTokenSecret(ctx)
	}

	return nil
}

func (r *Reconciler) ensureAuthTokenSecret(ctx context.Context) error {
	agSecretData, err := r.getActiveGateAuthToken()
	if err != nil {
		return errors.WithMessagef(err, "failed to create secret '%s'", r.dynakube.ActiveGateAuthTokenSecret())
	}
	return r.createSecret(ctx, agSecretData)
}

func (r *Reconciler) getActiveGateAuthToken() (map[string][]byte, error) {
	authTokenInfo, err := r.dtc.GetActiveGateAuthToken(r.dynakube.Name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return map[string][]byte{
		ActiveGateAuthTokenName: []byte(authTokenInfo.Token),
	}, nil
}

func (r *Reconciler) createSecret(ctx context.Context, secretData map[string][]byte) error {
	secretName := r.dynakube.ActiveGateAuthTokenSecret()
	secret, err := kubeobjects.CreateSecret(r.scheme, r.dynakube,
		kubeobjects.NewSecretNameModifier(secretName),
		kubeobjects.NewSecretNamespaceModifier(r.dynakube.Namespace),
		kubeobjects.NewSecretDataModifier(secretData))
	if err != nil {
		return errors.WithStack(err)
	}

	err = r.client.Create(ctx, secret)
	if err != nil {
		return errors.Errorf("failed to create secret '%s': %v", secretName, err)
	}
	return nil
}

func (r *Reconciler) deleteSecret(ctx context.Context, secret *corev1.Secret) error {
	if err := r.client.Delete(ctx, secret); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}

func isSecretOutdated(secret *corev1.Secret) bool {
	return secret.CreationTimestamp.Add(AuthTokenRotationInterval).Before(time.Now())
}
