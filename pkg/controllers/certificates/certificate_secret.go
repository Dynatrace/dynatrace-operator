package certificates

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type certificateSecret struct {
	secret          *corev1.Secret
	certificates    *Certs
	owner           *appsv1.Deployment
	existsInCluster bool
}

func newCertificateSecret(deployment *appsv1.Deployment) *certificateSecret {
	return &certificateSecret{
		owner: deployment,
	}
}

func (certSecret *certificateSecret) setSecretFromReader(ctx context.Context, apiReader client.Reader, namespace string) error {
	secrets := k8ssecret.Query(nil, apiReader, log)
	secret, err := secrets.Get(ctx, types.NamespacedName{Name: buildSecretName(), Namespace: namespace})

	switch {
	case k8serrors.IsNotFound(err):
		certSecret.secret, err = k8ssecret.Build(certSecret.owner,
			buildSecretName(),
			map[string][]byte{})
		if err != nil {
			return errors.WithStack(err)
		}

		certSecret.existsInCluster = false
	case err != nil:
		return errors.WithStack(err)
	default:
		certSecret.secret = secret
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
	default:
		return true
	}
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

func (certSecret *certificateSecret) isCRDConversionValid(crd *apiextensionv1.CustomResourceDefinition) bool {
	return !hasConversionWebhook(*crd) || certSecret.isBundleValid(crd.Spec.Conversion.Webhook.ClientConfig.CABundle)
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
