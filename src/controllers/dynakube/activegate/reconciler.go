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
	AGSecretSuffix             = "-activegate-tenant-secret"
	CommunicationEndpointsName = "communication-endpoints"
	TenantTokenName            = "tenant-token"
	TenantUuidName             = "tenant-uuid"
)

type Reconciler struct {
	client.Client
	apiReader client.Reader
	instance  *dynatracev1beta1.DynaKube
	scheme    *runtime.Scheme
	apiToken  string
	dtc       dtclient.Client
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, instance *dynatracev1beta1.DynaKube, apiToken string, dtc dtclient.Client) *Reconciler {
	return &Reconciler{
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		instance:  instance,
		apiToken:  apiToken,
		dtc:       dtc,
	}
}

func (r *Reconciler) Reconcile() error {
	if r.instance.Spec.ActiveGate.AGTenantSecret == "" {
		err := r.reconcileActiveGateSecret()
		if err != nil {
			log.Error(err, "could not reconcile ActiveGate tenant secret")
			return errors.WithStack(err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileActiveGateSecret() error {
	agSecretData, err := r.getActiveGateTenantInfo()
	if err != nil {
		return fmt.Errorf("could not generate ag secret data: %w", err)
	}

	agSecret, err := r.createAGSecretIfNotExists(agSecretData)
	if err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}

	return r.updateAGSecretIfOutdated(agSecret, agSecretData)
}

func (r *Reconciler) getActiveGateTenantInfo() (map[string][]byte, error) {
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

func (r *Reconciler) createAGSecretIfNotExists(agSecretData map[string][]byte) (*corev1.Secret, error) {
	var config corev1.Secret
	err := r.apiReader.Get(context.TODO(), client.ObjectKey{Name: extendWithAGSecretSuffix(r.instance.Name), Namespace: r.instance.Namespace}, &config)
	if k8serrors.IsNotFound(err) {
		log.Info("creating ag secret")
		return r.createAGSecret(agSecretData)
	}
	return &config, err
}

func (r *Reconciler) updateAGSecretIfOutdated(agSecret *corev1.Secret, desiredAGSecretData map[string][]byte) error {
	if !kubeobjects.IsSecretEqual(agSecret, desiredAGSecretData) {
		return r.updateAGSecret(agSecret, desiredAGSecretData)
	}
	return nil
}

func (r *Reconciler) createAGSecret(agSecretData map[string][]byte) (*corev1.Secret, error) {
	agSecret := BuildAGSecret(r.instance, agSecretData)

	if err := controllerutil.SetControllerReference(r.instance, agSecret, r.scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	err := r.Create(context.TODO(), agSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret '%s': %w", extendWithAGSecretSuffix(r.instance.Name), err)
	}
	return agSecret, nil
}

func (r *Reconciler) updateAGSecret(agSecret *corev1.Secret, desiredAGSecretData map[string][]byte) error {
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
	return name + AGSecretSuffix
}
