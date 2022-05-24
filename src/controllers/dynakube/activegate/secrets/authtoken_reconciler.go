package secrets

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
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
)

type AuthTokenReconciler struct {
	client.Client
	apiReader client.Reader
	instance  *dynatracev1beta1.DynaKube
	scheme    *runtime.Scheme
	apiToken  string
	dtc       dtclient.Client
}

func NewAuthTokenReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, instance *dynatracev1beta1.DynaKube, apiToken string, dtc dtclient.Client) *AuthTokenReconciler {
	return &AuthTokenReconciler{
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		instance:  instance,
		apiToken:  apiToken,
		dtc:       dtc,
	}
}

func (r *AuthTokenReconciler) Reconcile() error {
	_, err := r.createSecretIfNotExists()
	if err != nil {
		return errors.Errorf("failed to create activeGateAuthToken secret: %v", err)
	}

	return nil
}

func (r *AuthTokenReconciler) createSecretIfNotExists() (*corev1.Secret, error) {
	var config corev1.Secret
	err := r.apiReader.Get(context.TODO(),
		client.ObjectKey{Name: r.instance.ActiveGateAuthTokenSecret(), Namespace: r.instance.Namespace},
		&config)
	if k8serrors.IsNotFound(err) {
		log.Info("creating activeGateAuthToken secret")
		agSecretData, err := r.getActiveGateAuthToken()
		if err != nil {
			return nil, errors.Errorf("failed to create secret '%s': %v", extendWithAGSecretSuffix(r.instance.Name), err)
		}
		return r.createSecret(agSecretData)
	}
	return &config, errors.WithStack(err)
}

func (r *AuthTokenReconciler) getActiveGateAuthToken() (map[string][]byte, error) {
	authTokenInfo, err := r.dtc.GetActiveGateAuthToken(r.instance.Name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return map[string][]byte{
		ActiveGateAuthTokenName: []byte(authTokenInfo.Token),
	}, nil
}

func (r *AuthTokenReconciler) createSecret(secretData map[string][]byte) (*corev1.Secret, error) {
	secret := kubeobjects.CreateSecret(r.instance.ActiveGateAuthTokenSecret(), r.instance.Namespace, secretData)
	if err := controllerutil.SetControllerReference(r.instance, secret, r.scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	err := r.Create(context.TODO(), secret)
	if err != nil {
		return nil, errors.Errorf("failed to create secret '%s': %v", extendWithAGSecretSuffix(r.instance.Name), err)
	}
	return secret, nil
}
