package certificates

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type certificateSecret struct {
	secret          *corev1.Secret
	certificates    *Certs
	existsInCluster bool
}

func newCertificateSecret() *certificateSecret {
	return &certificateSecret{}
}

func (certSecret *certificateSecret) setSecretFromReader(ctx context.Context, apiReader client.Reader, namespace string) error {
	// Not linting this line because a lot has already been refactored in the pr this command was added
	// Should be refactored whenever someone reads this comment
	//nolint:staticcheck
	secret, err := kubeobjects.GetSecret(ctx, apiReader, buildSecretName(), namespace)
	if err != nil {
		return errors.WithStack(err)
	}
	if secret == nil {
		secret = kubeobjects.NewSecret(buildSecretName(), namespace, map[string][]byte{})
		certSecret.existsInCluster = false
	} else {
		certSecret.existsInCluster = true
	}

	certSecret.secret = secret
	return nil
}

func (certSecret *certificateSecret) isRecent() bool {
	if certSecret.secret == nil && certSecret.certificates == nil {
		return true
	} else if certSecret.secret == nil || certSecret.certificates == nil {
		return false
	} else if !reflect.DeepEqual(certSecret.certificates.Data, certSecret.secret.Data) {
		// certificates need to be updated
		return false
	}
	return true
}

func (certSecret *certificateSecret) validateCertificates(namespace string) error {
	certs := Certs{
		Domain:  getDomain(namespace),
		SrcData: certSecret.secret.Data,
		Now:     time.Now(),
	}
	if err := certs.ValidateCerts(); err != nil {
		return errors.WithStack(err)
	}

	certSecret.certificates = &certs

	return nil
}

func buildSecretName() string {
	return fmt.Sprintf("%s%s", webhook.DeploymentName, secretPostfix)
}

func getDomain(namespace string) string {
	return fmt.Sprintf("%s.%s.svc", webhook.DeploymentName, namespace)
}

func (certSecret *certificateSecret) areConfigsValid(configs []*admissionregistrationv1.WebhookClientConfig) bool {
	for i := range configs {
		if configs[i] != nil && !certSecret.isClientConfigValid(*configs[i]) {
			return false
		}
	}
	return true
}

func (certSecret *certificateSecret) isClientConfigValid(clientConfig admissionregistrationv1.WebhookClientConfig) bool {
	return len(clientConfig.CABundle) != 0 && bytes.Equal(clientConfig.CABundle, certSecret.certificates.Data[RootCert])
}

func (certSecret *certificateSecret) createOrUpdateIfNecessary(ctx context.Context, clt client.Client) error {
	if certSecret.isRecent() && certSecret.existsInCluster {
		return nil
	}

	var err error

	certSecret.secret.Data = certSecret.certificates.Data
	if certSecret.existsInCluster {
		err = clt.Update(ctx, certSecret.secret)
		log.Info("updated certificates secret")
	} else {
		err = clt.Create(ctx, certSecret.secret)
		log.Info("created certificates secret")
	}

	return errors.WithStack(err)
}

func (certSecret *certificateSecret) updateClientConfigurations(ctx context.Context, clt client.Client, webhookClientConfigs []*admissionregistrationv1.WebhookClientConfig, webhookConfig client.Object) error {
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
