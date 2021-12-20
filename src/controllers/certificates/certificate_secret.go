package certificates

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	v1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"github.com/pkg/errors"
)

type certificateSecret struct {
	secret          *corev1.Secret
	certificates    *Certs
	existsInCluster bool
	isRecent        bool
}

func createCertificateSecret(apiReader client.Reader, ctx context.Context, namespace string) (*certificateSecret, error) {
	certSecret := &certificateSecret{}
	secret, err := findSecret(ctx, apiReader, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if secret == nil {
		secret = emptySecret(namespace)
		certSecret.existsInCluster = false
	} else {
		certSecret.existsInCluster = true
	}

	certSecret.secret = secret
	certSecret.certificates, err = certSecret.validateCertificates(namespace)

	if !reflect.DeepEqual(certSecret.certificates.Data, certSecret.secret.Data) {
		// certificates need to be updated
		certSecret.secret.Data = certSecret.certificates.Data
		certSecret.isRecent = false
	} else {
		certSecret.isRecent = true
	}
	return certSecret, errors.WithStack(err)
}

func findSecret(ctx context.Context, apiReader client.Reader, namespace string) (*corev1.Secret, error) {
	var oldSecret corev1.Secret
	err := apiReader.Get(ctx, client.ObjectKey{Name: buildSecretName(), Namespace: namespace}, &oldSecret)
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	return &oldSecret, errors.WithStack(err)
}

func emptySecret(namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildSecretName(),
			Namespace: namespace,
		},
		Data: map[string][]byte{},
	}
}

func (certSecret *certificateSecret) validateCertificates(namespace string) (*Certs, error) {
	certs := Certs{
		Domain:  getDomain(namespace),
		SrcData: certSecret.secret.Data,
		Now:     time.Now(),
	}
	if err := certs.ValidateCerts(); err != nil {
		return nil, errors.WithStack(err)
	}
	return &certs, nil
}

func buildSecretName() string {
	return fmt.Sprintf("%s%s", webhook.DeploymentName, secretPostfix)
}

func getDomain(namespace string) string {
	return fmt.Sprintf("%s.%s.svc", webhook.DeploymentName, namespace)
}

func (certSecret *certificateSecret) areConfigsValid(configs []*v1.WebhookClientConfig) bool {
	for i := range configs {
		if !certSecret.isClientConfigValid(*configs[i]) {
			return false
		}
	}
	return true
}

func (certSecret *certificateSecret) isClientConfigValid(clientConfig v1.WebhookClientConfig) bool {
	return len(clientConfig.CABundle) != 0 && bytes.Equal(clientConfig.CABundle, certSecret.certificates.Data[RootCert])
}

func (certSecret *certificateSecret) isOutdated() bool {
	return !certSecret.isRecent
}

func (certSecret *certificateSecret) createOrUpdateIfNecessary(ctx context.Context, clt client.Client) error {
	if certSecret.isRecent {
		return nil
	}

	var err error
	if certSecret.existsInCluster {
		err = clt.Update(ctx, certSecret.secret)
		log.Info("created certificates secret")
	} else {
		err = clt.Create(ctx, certSecret.secret)
		log.Info("updated certificates secret")
	}

	return errors.WithStack(err)
}

func (certSecret *certificateSecret) updateClientConfigurations(ctx context.Context, clt client.Client, webhookClientConfigs []*v1.WebhookClientConfig, webhookConfig client.Object) error {
	if webhookConfig == nil || reflect.ValueOf(webhookConfig).IsNil() {
		return nil
	}

	data, hasData := certSecret.secret.Data[RootCert]
	if !hasData {
		return errors.New(errorCertificatesSecretEmpty)
	}

	if oldData, hasOldData := certSecret.secret.Data[RootCertOld]; hasOldData {
		data = append(data, oldData...)
	}

	for i := range webhookClientConfigs {
		webhookClientConfigs[i].CABundle = data
	}

	return errors.WithStack(clt.Update(ctx, webhookConfig))
}
