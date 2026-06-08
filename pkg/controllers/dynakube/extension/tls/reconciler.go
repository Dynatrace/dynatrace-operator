package tls

import (
	"context"
	"crypto/x509"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	extensionsSelfSignedTLSCommonNameSuffix = "extension-controller"
)

type reconciler struct {
	timeProvider *timeprovider.Provider
	secrets      k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader) *reconciler {
	return &reconciler{
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(clt, apiReader),
	}
}

func (r *reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, _ = logd.NewFromContext(ctx, "extension-tls")

	if ext := dk.Extensions(); ext.IsAnyEnabled() && ext.NeedsSelfSignedTLS() {
		return r.reconcileSelfSignedTLSSecret(ctx, dk)
	}

	if meta.FindStatusCondition(*dk.Conditions(), conditionType) == nil {
		return nil
	}

	defer func() {
		meta.RemoveStatusCondition(dk.Conditions(), conditionType)
	}()

	return r.deleteSelfSignedTLSSecret(ctx, dk)
}

func (r *reconciler) reconcileSelfSignedTLSSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	_, err := r.secrets.Get(ctx, types.NamespacedName{
		Name:      dk.Extensions().GetSelfSignedTLSSecretName(),
		Namespace: dk.Namespace,
	})
	if err != nil && k8serrors.IsNotFound(err) {
		return r.createSelfSignedTLSSecret(ctx, dk)
	}

	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), conditionType, err)

		return err
	}

	return nil
}

func (r *reconciler) deleteSelfSignedTLSSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	return r.secrets.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.Extensions().GetSelfSignedTLSSecretName(),
			Namespace: dk.Namespace,
		},
	})
}

func (r *reconciler) createSelfSignedTLSSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	cert, err := certificates.New(r.timeProvider)
	if err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	cert.Cert.DNSNames = certificates.AltNames(dk.Name, dk.Namespace, extensionsSelfSignedTLSCommonNameSuffix)
	cert.Cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
	cert.Cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	cert.Cert.Subject.CommonName = certificates.CommonName(dk.Name, dk.Namespace, extensionsSelfSignedTLSCommonNameSuffix)

	err = cert.SelfSign()
	if err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	pemCert, pemPk, err := cert.ToPEM()
	if err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.ExtensionComponentLabel)
	secretData := map[string][]byte{consts.TLSCrtDataName: pemCert, consts.TLSKeyDataName: pemPk}

	secret, err := k8ssecret.Build(dk, dk.Extensions().GetSelfSignedTLSSecretName(), secretData, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	secret.Type = corev1.SecretTypeTLS

	err = r.secrets.Create(ctx, secret)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), conditionType, err)

		return err
	}

	k8sconditions.SetSecretCreated(dk.Conditions(), conditionType, secret.Name)

	return nil
}
