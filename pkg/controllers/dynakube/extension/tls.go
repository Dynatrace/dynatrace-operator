package extension

import (
	"context"
	"crypto/x509"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *reconciler) reconcileTLSSecret(ctx context.Context) error {
	query := k8ssecret.Query(r.client, r.client, log)

	if !r.dk.ExtensionsNeedsSelfSignedTLS() {
		return query.Delete(ctx, &corev1.Secret{ObjectMeta: v1.ObjectMeta{Name: getSelfSignedTLSSecretName(r.dk.Name), Namespace: r.dk.Namespace}})
	}

	secret, err := query.Get(ctx, types.NamespacedName{
		Name:      getSelfSignedTLSSecretName(r.dk.Name),
		Namespace: r.dk.Namespace,
	})

	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if k8serrors.IsNotFound(err) {
		err = r.createOrUpdateTLSSecret(ctx)
		if err != nil {
			return err
		}

		return nil
	}

	err = r.reconcileTLSSecretExpiration(ctx, secret)
	if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) createOrUpdateTLSSecret(ctx context.Context) error {
	cert, err := certificates.New(r.timeProvider)
	if err != nil {
		return err
	}

	cert.Cert.DNSNames = getCertificateAltNames(r.dk.Name)
	cert.Cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
	cert.Cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	cert.Cert.Subject.CommonName = r.dk.Name + consts.ExtensionsSelfSignedTLSCommonNameSuffix

	err = cert.SelfSign()
	if err != nil {
		return err
	}

	pemCert, pemPk, err := cert.ToPEM()
	if err != nil {
		return err
	}

	coreLabels := k8slabels.NewCoreLabels(r.dk.Name, k8slabels.ExtensionComponentLabel)
	secretData := map[string][]byte{consts.TLSCrtDataName: pemCert, consts.TLSKeyDataName: pemPk}

	secret, err := k8ssecret.Build(r.dk, getSelfSignedTLSSecretName(r.dk.Name), secretData, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		return err
	}

	secret.Type = corev1.SecretTypeTLS

	query := k8ssecret.Query(r.client, r.client, log)

	_, err = query.CreateOrUpdate(ctx, secret)
	if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) reconcileTLSSecretExpiration(ctx context.Context, secret *corev1.Secret) error {
	isValid, err := certificates.ValidateCertificateExpiration(secret.Data[consts.TLSCrtDataName], consts.ExtensionsSelfSignedTLSRenewalThreshold, r.timeProvider.Now().Time, log)
	if err != nil || !isValid {
		log.Info("server certificate failed to parse or is outdated")

		return r.createOrUpdateTLSSecret(ctx)
	}

	return nil
}

func getCertificateAltNames(dkName string) []string {
	return []string{
		dkName + "-extensions-controller.dynatrace",
		dkName + "-extensions-controller.dynatrace.svc",
		dkName + "-extensions-controller.dynatrace.svc.cluster.local",
	}
}

func getSelfSignedTLSSecretName(dkName string) string {
	return dkName + consts.ExtensionsSelfSignedTLSSecretSuffix
}
