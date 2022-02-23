package activegate

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	CommunicationEndpointsName = "communication-endpoints"
	TenantTokenName            = "tenant-token"
	TenantUuidName             = "tenant-uuid"
)

type TenantSecretReconciler struct {
	client.Client
	apiReader client.Reader
	instance  *dynatracev1beta1.DynaKube
	scheme    *runtime.Scheme
	apiToken  string
	dtc       dtclient.Client
}

func NewTenantSecretReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, instance *dynatracev1beta1.DynaKube, apiToken string, dtc dtclient.Client) *TenantSecretReconciler {
	return &TenantSecretReconciler{
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		instance:  instance,
		apiToken:  apiToken,
		dtc:       dtc,
	}
}

func (r *TenantSecretReconciler) Reconcile() error {
	err := r.reconcileSecret()
	if err != nil {
		log.Error(err, "could not reconcile ActiveGate tenant secret")
		return errors.WithStack(err)
	}

	return nil
}

func (r *TenantSecretReconciler) reconcileSecret() error {
	agSecretData, err := r.getActiveGateTenantInfo()
	if err != nil {
		return fmt.Errorf("failed to fetch ActiveGate tenant info: %w", err)
	}

	agSecret, err := r.createSecretIfNotExists(agSecretData)
	if err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}

	return r.updateSecretIfOutdated(agSecret, agSecretData)
}

func (r *TenantSecretReconciler) getActiveGateTenantInfo() (map[string][]byte, error) {
	tenantInfo, err := r.dtc.GetActiveGateTenantInfo()

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return map[string][]byte{
		TenantUuidName:             []byte(tenantInfo.UUID),
		TenantTokenName:            []byte(tenantInfo.Token),
		CommunicationEndpointsName: []byte(tenantInfo.Endpoints),
	}, nil
}

func (r *TenantSecretReconciler) createSecretIfNotExists(agSecretData map[string][]byte) (*corev1.Secret, error) {
	var config corev1.Secret
	err := r.apiReader.Get(context.TODO(),
		client.ObjectKey{Name: extendWithAGSecretSuffix(r.instance.Name), Namespace: r.instance.Namespace},
		&config)
	if k8serrors.IsNotFound(err) {
		log.Info("creating ag secret")
		return r.createSecret(agSecretData)
	}
	return &config, err
}

func (r *TenantSecretReconciler) updateSecretIfOutdated(secret *corev1.Secret, desiredSecret map[string][]byte) error {
	if !kubeobjects.IsSecretEqual(secret, desiredSecret) {
		return r.updateSecret(secret, desiredSecret)
	}
	return nil
}

func (r *TenantSecretReconciler) createSecret(secretData map[string][]byte) (*corev1.Secret, error) {
	secret := BuildAGSecret(r.instance, secretData)

	if err := controllerutil.SetControllerReference(r.instance, secret, r.scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	err := r.Create(context.TODO(), secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret '%s': %w", extendWithAGSecretSuffix(r.instance.Name), err)
	}
	return secret, nil
}

func (r *TenantSecretReconciler) updateSecret(agSecret *corev1.Secret, desiredAGSecretData map[string][]byte) error {
	log.Info("updating secret", "name", agSecret.Name)
	agSecret.Data = desiredAGSecretData
	if err := r.Update(context.TODO(), agSecret); err != nil {
		return fmt.Errorf("failed to update secret %s: %w", agSecret.Name, err)
	}
	return nil
}

func BuildAGSecret(instance *dynatracev1beta1.DynaKube, agSecretData map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      extendWithAGSecretSuffix(instance.Name),
			Namespace: instance.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: agSecretData,
	}
}

func extendWithAGSecretSuffix(name string) string {
	return name + dynatracev1beta1.TenantSecretSuffix
}
