package authtoken

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func (r *Reconciler) Reconcile() error {
	if !dynatracev1beta1.IsRequestOutdated(r.dynakube.Status.DynatraceApi.LastAuthTokenSecretUpdate) {
		log.Info(r.dynakube.Status.DynatraceApi.NotOutdatedMessage("ActiveGate auth token secret generation"))
		return nil
	}

	err := r.reconcileAuthTokenSecret()
	if err != nil {
		return errors.Errorf("failed to create activeGateAuthToken secret: %v", err)
	}

	r.dynakube.Status.DynatraceApi.LastAuthTokenSecretUpdate = metav1.Now()
	return nil
}

func (r *Reconciler) reconcileAuthTokenSecret() error {
	var secret corev1.Secret
	err := r.apiReader.Get(context.TODO(),
		client.ObjectKey{Name: r.dynakube.ActiveGateAuthTokenSecret(), Namespace: r.dynakube.Namespace},
		&secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating activeGateAuthToken secret")
			return r.ensureAuthTokenSecret()
		}
		return errors.WithStack(err)
	}
	if isSecretOutdated(&secret) {
		log.Info("activeGateAuthToken is outdated, creating new one")
		if err := r.deleteSecret(&secret); err != nil {
			return errors.WithStack(err)
		}
		return r.ensureAuthTokenSecret()
	}

	return nil
}

func (r *Reconciler) ensureAuthTokenSecret() error {
	agSecretData, err := r.getActiveGateAuthToken()
	if err != nil {
		return errors.Errorf("failed to create secret '%s': %v", r.dynakube.ActiveGateAuthTokenSecret(), err)
	}
	return r.createSecret(agSecretData)
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

func (r *Reconciler) createSecret(secretData map[string][]byte) error {
	secretName := r.dynakube.ActiveGateAuthTokenSecret()
	secret := kubeobjects.NewSecret(secretName, r.dynakube.Namespace, secretData)
	if err := controllerutil.SetControllerReference(r.dynakube, secret, r.scheme); err != nil {
		return errors.WithStack(err)
	}

	err := r.client.Create(context.TODO(), secret)
	if err != nil {
		return errors.Errorf("failed to create secret '%s': %v", secretName, err)
	}
	return nil
}

func (r *Reconciler) deleteSecret(secret *corev1.Secret) error {
	if err := r.client.Delete(context.TODO(), secret); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}

func isSecretOutdated(secret *corev1.Secret) bool {
	return secret.CreationTimestamp.Add(AuthTokenRotationInterval).Before(time.Now())
}
