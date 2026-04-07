package tls

import (
	"context"
	"crypto/x509"
	"net"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	activeGateSelfSignedTLSCommonNameSuffix = "activegate"

	tlsCrtDataName = "server.crt"
)

type Reconciler struct {
	timeProvider *timeprovider.Provider
	secrets      k8ssecret.QueryObject
}

func NewReconciler(client client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(client, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	if dk.ActiveGate().IsEnabled() && dk.ActiveGate().IsAutomaticTLSSecretEnabled() && dk.ActiveGate().TLSSecretName == "" {
		return r.reconcileSelfSignedTLSSecret(ctx, dk)
	}

	if meta.FindStatusCondition(*dk.Conditions(), conditionType) == nil {
		return nil
	}
	defer meta.RemoveStatusCondition(dk.Conditions(), conditionType)

	return r.deleteSelfSignedTLSSecret(ctx, dk)
}

func (r *Reconciler) reconcileSelfSignedTLSSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	_, err := r.secrets.Get(ctx, types.NamespacedName{
		Name:      dk.ActiveGate().GetTLSSecretName(),
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

func (r *Reconciler) deleteSelfSignedTLSSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	err := r.secrets.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.ActiveGate().GetAutoTLSSecretName(),
			Namespace: dk.Namespace,
		},
	})

	if k8serrors.IsNotFound(err) {
		return nil
	}

	return err
}

func (r *Reconciler) createSelfSignedTLSSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	cert, err := certificates.New(r.timeProvider)
	if err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	cert.Cert.DNSNames = certificates.AltNames(dk.Name, dk.Namespace, activeGateSelfSignedTLSCommonNameSuffix)
	cert.Cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
	cert.Cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	cert.Cert.Subject.CommonName = certificates.CommonName(dk.Name, dk.Namespace, activeGateSelfSignedTLSCommonNameSuffix)

	ipAddresses, err := getCertificateAltIPs(dk.Status.ActiveGate.ServiceIPs)
	if err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	cert.Cert.IPAddresses = ipAddresses

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

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.ActiveGateComponentLabel)
	secretData := map[string][]byte{
		consts.TLSCrtDataName: pemCert,
		consts.TLSKeyDataName: pemPk,
		tlsCrtDataName:        pemCert,
	}

	secret, err := k8ssecret.Build(dk, dk.ActiveGate().GetTLSSecretName(), secretData, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	secret.Type = corev1.SecretTypeOpaque

	err = r.secrets.Create(ctx, secret)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), conditionType, err)

		return err
	}

	k8sconditions.SetSecretCreated(dk.Conditions(), conditionType, secret.Name)

	return nil
}

func getCertificateAltIPs(ips []string) ([]net.IP, error) {
	altIPs := []net.IP{}

	for _, ip := range ips {
		netIP := net.ParseIP(ip)
		if netIP == nil {
			return nil, errors.Errorf("failed to parse '%s' IP address", ip)
		}

		altIPs = append(altIPs, netIP)
	}

	return altIPs, nil
}
