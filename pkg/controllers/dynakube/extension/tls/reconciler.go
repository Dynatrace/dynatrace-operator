package tls

import (
	"context"
	"crypto/x509"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client       client.Client
	apiReader    client.Reader
	timeProvider *timeprovider.Provider

	dk *dynakube.DynaKube
}

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:       clt,
		apiReader:    apiReader,
		dk:           dk,
		timeProvider: timeprovider.New(),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	if r.dk.IsExtensionsEnabled() && r.dk.ExtensionsNeedsSelfSignedTLS() {
		return r.reconcileSelfSignedTLSSecret(ctx)
	}

	return r.deleteSelfSignedTLSSecret(ctx)
}

func (r *reconciler) reconcileSelfSignedTLSSecret(ctx context.Context) error {
	query := k8ssecret.Query(r.client, r.client, log)

	_, err := query.Get(ctx, types.NamespacedName{
		Name:      getSelfSignedTLSSecretName(r.dk.Name),
		Namespace: r.dk.Namespace,
	})

	if err != nil && k8serrors.IsNotFound(err) {
		return r.createSelfSignedTLSSecret(ctx)
	}

	return err
}

func (r *reconciler) deleteSelfSignedTLSSecret(ctx context.Context) error {
	query := k8ssecret.Query(r.client, r.client, log)

	return query.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSelfSignedTLSSecretName(r.dk.Name),
			Namespace: r.dk.Namespace,
		},
	})
}

func (r *reconciler) createSelfSignedTLSSecret(ctx context.Context) error {
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

	err = query.Create(ctx, secret)
	if err != nil {
		return err
	}

	return nil
}

func GetTLSSecretName(dk *dynakube.DynaKube) string {
	if dk.ExtensionsNeedsSelfSignedTLS() {
		return getSelfSignedTLSSecretName(dk.Name)
	}

	return dk.ExtensionsTLSRefName()
}

func getSelfSignedTLSSecretName(dkName string) string {
	return dkName + consts.ExtensionsSelfSignedTLSSecretSuffix
}

func getCertificateAltNames(dkName string) []string {
	return []string{
		dkName + "-extensions-controller.dynatrace",
		dkName + "-extensions-controller.dynatrace.svc",
		dkName + "-extensions-controller.dynatrace.svc.cluster.local",
	}
}
