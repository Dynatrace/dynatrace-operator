package tls

import (
	"context"
	"crypto/x509"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
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
	var secretHash string

	var err error

	if r.dk.ExtensionsNeedsSelfSignedTLS() {
		secretHash, err = r.reconcileSelfSignedMode(ctx)
		if err != nil {
			return err
		}
	} else {
		secretHash, err = r.reconcileTLSRefNameMode(ctx)
		if err != nil {
			return err
		}
	}

	err = r.reconcileStsSecretHash(ctx, dynakube.ExtensionsExecutionControllerStatefulsetName, secretHash)
	if err != nil {
		return err
	}

	err = r.reconcileStsSecretHash(ctx, dynakube.ExtensionsCollectorStatefulsetName, secretHash)
	if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) reconcileSelfSignedMode(ctx context.Context) (secretHash string, err error) {
	secret, err := k8ssecret.Query(r.client, r.client, log).Get(ctx, types.NamespacedName{
		Name:      getSelfSignedTLSSecretName(r.dk.Name),
		Namespace: r.dk.Namespace,
	})

	if err != nil && k8serrors.IsNotFound(err) {
		return r.createSelfSignedTLSSecret(ctx)
	}

	if err != nil {
		return "", err
	}

	return hasher.GenerateHash(secret.Data)
}

func (r *reconciler) reconcileTLSRefNameMode(ctx context.Context) (secretHash string, err error) {
	query := k8ssecret.Query(r.client, r.client, log)

	err = query.Delete(ctx, &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      getSelfSignedTLSSecretName(r.dk.Name),
			Namespace: r.dk.Namespace,
		},
	})
	if err != nil {
		return "", err
	}

	secret, err := query.Get(ctx, types.NamespacedName{
		Name:      r.dk.ExtensionsTLSRefName(),
		Namespace: r.dk.Namespace,
	})
	if err != nil {
		return "", err
	}

	return hasher.GenerateHash(secret.Data)
}

func (r *reconciler) createSelfSignedTLSSecret(ctx context.Context) (hash string, err error) {
	cert, err := certificates.New(r.timeProvider)
	if err != nil {
		return "", err
	}

	cert.Cert.DNSNames = getCertificateAltNames(r.dk.Name)
	cert.Cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
	cert.Cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	cert.Cert.Subject.CommonName = r.dk.Name + consts.ExtensionsSelfSignedTLSCommonNameSuffix

	err = cert.SelfSign()
	if err != nil {
		return "", err
	}

	pemCert, pemPk, err := cert.ToPEM()
	if err != nil {
		return "", err
	}

	coreLabels := k8slabels.NewCoreLabels(r.dk.Name, k8slabels.ExtensionComponentLabel)
	secretData := map[string][]byte{consts.TLSCrtDataName: pemCert, consts.TLSKeyDataName: pemPk}

	secret, err := k8ssecret.Build(r.dk, getSelfSignedTLSSecretName(r.dk.Name), secretData, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		return "", err
	}

	secret.Type = corev1.SecretTypeTLS

	query := k8ssecret.Query(r.client, r.client, log)

	err = query.Create(ctx, secret)
	if err != nil {
		return "", err
	}

	return hasher.GenerateHash(secret.Data)
}

func (r *reconciler) reconcileStsSecretHash(ctx context.Context, stsName string, secretHash string) error {
	query := statefulset.Query(r.client, r.apiReader, log)

	sts, err := query.Get(ctx, types.NamespacedName{Name: stsName, Namespace: r.dk.Namespace})
	if k8serrors.IsNotFound(err) {
		// no need to reconcile - sts is not created yet
		return nil
	}

	if err != nil {
		return err
	}

	if sts.Spec.Template.Annotations[consts.ExtensionsAnnotationSecretHash] != secretHash {
		sts.Spec.Template.Annotations[consts.ExtensionsAnnotationSecretHash] = secretHash

		return query.Update(ctx, sts)
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
