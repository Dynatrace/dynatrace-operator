package tls

import (
	"context"
	"crypto/x509"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
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
	dk           *dynakube.DynaKube
	secrets      k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		dk:           dk,
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(clt, apiReader, log),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	if ext := r.dk.Extensions(); ext.IsAnyEnabled() && ext.NeedsSelfSignedTLS() {
		return r.reconcileSelfSignedTLSSecret(ctx)
	}

	if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
		return nil
	}
	defer meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

	return r.deleteSelfSignedTLSSecret(ctx)
}

func (r *reconciler) reconcileSelfSignedTLSSecret(ctx context.Context) error {
	_, err := r.secrets.Get(ctx, types.NamespacedName{
		Name:      r.dk.Extensions().GetSelfSignedTLSSecretName(),
		Namespace: r.dk.Namespace,
	})
	if err != nil && k8serrors.IsNotFound(err) {
		return r.createSelfSignedTLSSecret(ctx)
	}

	if err != nil {
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	return nil
}

func (r *reconciler) deleteSelfSignedTLSSecret(ctx context.Context) error {
	return r.secrets.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.dk.Extensions().GetSelfSignedTLSSecretName(),
			Namespace: r.dk.Namespace,
		},
	})
}

func (r *reconciler) createSelfSignedTLSSecret(ctx context.Context) error {
	cert, err := certificates.New(r.timeProvider)
	if err != nil {
		k8sconditions.SetSecretGenFailed(r.dk.Conditions(), conditionType, err)

		return err
	}

	cert.Cert.DNSNames = certificates.AltNames(r.dk.Name, r.dk.Namespace, extensionsSelfSignedTLSCommonNameSuffix)
	cert.Cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
	cert.Cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	cert.Cert.Subject.CommonName = certificates.CommonName(r.dk.Name, r.dk.Namespace, extensionsSelfSignedTLSCommonNameSuffix)

	err = cert.SelfSign()
	if err != nil {
		k8sconditions.SetSecretGenFailed(r.dk.Conditions(), conditionType, err)

		return err
	}

	pemCert, pemPk, err := cert.ToPEM()
	if err != nil {
		k8sconditions.SetSecretGenFailed(r.dk.Conditions(), conditionType, err)

		return err
	}

	coreLabels := k8slabel.NewCoreLabels(r.dk.Name, k8slabel.ExtensionComponentLabel)
	secretData := map[string][]byte{consts.TLSCrtDataName: pemCert, consts.TLSKeyDataName: pemPk}

	secret, err := k8ssecret.Build(r.dk, r.dk.Extensions().GetSelfSignedTLSSecretName(), secretData, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		k8sconditions.SetSecretGenFailed(r.dk.Conditions(), conditionType, err)

		return err
	}

	secret.Type = corev1.SecretTypeTLS

	err = r.secrets.Create(ctx, secret)
	if err != nil {
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	k8sconditions.SetSecretCreated(r.dk.Conditions(), conditionType, secret.Name)

	return nil
}
