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
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	query := kubeobjects.NewSecretQuery(ctx, nil, apiReader, log)
	secret, err := query.Get(types.NamespacedName{Name: buildSecretName(), Namespace: namespace})

	switch {
	case k8serrors.IsNotFound(err):
		certSecret.secret = kubeobjects.NewSecret(buildSecretName(), namespace, map[string][]byte{})
		certSecret.existsInCluster = false
	case err != nil:
		return errors.WithStack(err)
	default:
		certSecret.secret = &secret
		certSecret.existsInCluster = true
	}

	return nil
}

func (certSecret *certificateSecret) isRecent() bool {
	switch {
	case certSecret.secret == nil && certSecret.certificates == nil:
		return true
	case certSecret.secret == nil || certSecret.certificates == nil:
		return false
	case !reflect.DeepEqual(certSecret.certificates.Data, certSecret.secret.Data):
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

func (certSecret *certificateSecret) areWebhookConfigsValid(configs []*admissionregistrationv1.WebhookClientConfig) bool {
	for i := range configs {
		if configs[i] != nil && !certSecret.isBundleValid(configs[i].CABundle) {
			return false
		}
	}
	return true
}

func (certSecret *certificateSecret) isCRDConversionValid(conversion *apiextensionv1.CustomResourceConversion) bool {
	return certSecret.isBundleValid(conversion.Webhook.ClientConfig.CABundle)
}

func (certSecret *certificateSecret) isBundleValid(bundle []byte) bool {
	return len(bundle) != 0 && bytes.Equal(bundle, certSecret.certificates.Data[RootCert])
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

func (certSecret *certificateSecret) loadCombinedBundle() ([]byte, error) {
	data, hasData := certSecret.secret.Data[RootCert]
	if !hasData {
		return nil, errors.New(errorCertificatesSecretEmpty)
	}

	if oldData, hasOldData := certSecret.secret.Data[RootCertOld]; hasOldData {
		data = append(data, oldData...)
	}
	return data, nil
}
