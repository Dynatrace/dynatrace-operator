package tokens

import (
	"context"
	"fmt"
	"reflect"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	SecretsName = "dynatrace-tokens"
)

type Reconciler struct {
	client.Client
	apiReader client.Reader
	instance  *dynatracev1alpha1.DynaKube
	dtc       dtclient.Client
	log       logr.Logger
	scheme    *runtime.Scheme
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, instance *dynatracev1alpha1.DynaKube, dtc dtclient.Client, log logr.Logger) *Reconciler {
	return &Reconciler{
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		instance:  instance,
		dtc:       dtc,
		log:       log,
	}
}

func (r *Reconciler) Reconcile() (bool, error) {
	if r.instance.Spec.AGTokensSecret == "" {
		err := r.reconcileAGTokens()
		if err != nil {
			r.log.Error(err, "could not reconcile ag tokens")
			return false, errors.WithStack(err)
		}
	}

	return false, nil
}

func (r *Reconciler) reconcileAGTokens() error {
	agTokens, err := r.getAGTokens()
	if err != nil {
		return fmt.Errorf("could not fetch AG tokens: %w", err)
	}

	agTokensSecret, err := r.createAGTokensSecretIfNotExists(agTokens)
	if err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}

	return r.updateAGTokensIfOutdated(agTokensSecret, agTokens)
}

func (r *Reconciler) createAGTokensSecretIfNotExists(secretData map[string][]byte) (*corev1.Secret, error) {
	var config corev1.Secret
	err := r.apiReader.Get(context.TODO(), client.ObjectKey{Name: SecretsName, Namespace: r.instance.Namespace}, &config)
	if k8serrors.IsNotFound(err) {
		r.log.Info("Creating AG Tokens secret")
		return r.createAGTokensSecret(secretData)
	}
	return &config, err
}

func (r *Reconciler) updateAGTokensIfOutdated(agTokens *corev1.Secret, agTokensSecretData map[string][]byte) error {
	if !isAGTokensSecretEqual(agTokens, agTokensSecretData) {
		return r.updateAGTokensSecret(agTokens, agTokensSecretData)
	}
	return nil
}

func (r *Reconciler) createAGTokensSecret(agTokensSecretData map[string][]byte) (*corev1.Secret, error) {
	agTokens := BuildAGTokensSecret(r.instance, agTokensSecretData)

	if err := controllerutil.SetControllerReference(r.instance, agTokens, r.scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	err := r.Create(context.TODO(), agTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret '%s': %w", SecretsName, err)
	}
	return agTokens, nil
}

func (r *Reconciler) updateAGTokensSecret(agTokensSecret *corev1.Secret, desiredAGTokensSecretData map[string][]byte) error {
	r.log.Info(fmt.Sprintf("Updating secret %s", agTokensSecret.Name))
	agTokensSecret.Data = desiredAGTokensSecretData
	if err := r.Update(context.TODO(), agTokensSecret); err != nil {
		return fmt.Errorf("failed to update secret %s: %w", agTokensSecret.Name, err)
	}
	return nil
}

func (r *Reconciler) getAGTokens() (map[string][]byte, error) {
	info, err := r.dtc.GetAGTenantInfo()
	if err != nil {
		return nil, err
	}

	return map[string][]byte{"tenant-token": []byte(info.Token)}, nil
}

func isAGTokensSecretEqual(currentSecret *corev1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}

func BuildAGTokensSecret(instance *dynatracev1alpha1.DynaKube, agTokensSecretData map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretsName,
			Namespace: instance.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: agTokensSecretData,
	}
}
