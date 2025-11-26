package tls

import (
	"context"
	"crypto/x509"
	"net"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
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
	dk           *dynakube.DynaKube
	timeProvider *timeprovider.Provider
	secrets      k8ssecret.QueryObject
}

type ReconcilerBuilder func(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler

func NewReconciler(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		dk:           dk,
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(client, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dk.ActiveGate().IsEnabled() && r.dk.ActiveGate().IsAutomaticTLSSecretEnabled() && r.dk.ActiveGate().TLSSecretName == "" {
		return r.reconcileSelfSignedTLSSecret(ctx)
	}

	if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
		return nil
	}
	defer meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

	return r.deleteSelfSignedTLSSecret(ctx)
}

func (r *Reconciler) reconcileSelfSignedTLSSecret(ctx context.Context) error {
	_, err := r.secrets.Get(ctx, types.NamespacedName{
		Name:      r.dk.ActiveGate().GetTLSSecretName(),
		Namespace: r.dk.Namespace,
	})
	if err != nil && k8serrors.IsNotFound(err) {
		return r.createSelfSignedTLSSecret(ctx)
	}

	if err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	return nil
}

func (r *Reconciler) deleteSelfSignedTLSSecret(ctx context.Context) error {
	return r.secrets.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.dk.ActiveGate().GetTLSSecretName(),
			Namespace: r.dk.Namespace,
		},
	})
}

func (r *Reconciler) createSelfSignedTLSSecret(ctx context.Context) error {
	cert, err := certificates.New(r.timeProvider)
	if err != nil {
		conditions.SetSecretGenFailed(r.dk.Conditions(), conditionType, err)

		return err
	}

	cert.Cert.DNSNames = certificates.AltNames(r.dk.Name, r.dk.Namespace, activeGateSelfSignedTLSCommonNameSuffix)
	cert.Cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
	cert.Cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	cert.Cert.Subject.CommonName = certificates.CommonName(r.dk.Name, r.dk.Namespace, activeGateSelfSignedTLSCommonNameSuffix)

	ipAddresses, err := getCertificateAltIPs(r.dk.Status.ActiveGate.ServiceIPs)
	if err != nil {
		conditions.SetSecretGenFailed(r.dk.Conditions(), conditionType, err)

		return err
	}

	cert.Cert.IPAddresses = ipAddresses

	err = cert.SelfSign()
	if err != nil {
		conditions.SetSecretGenFailed(r.dk.Conditions(), conditionType, err)

		return err
	}

	pemCert, pemPk, err := cert.ToPEM()
	if err != nil {
		conditions.SetSecretGenFailed(r.dk.Conditions(), conditionType, err)

		return err
	}

	coreLabels := k8slabel.NewCoreLabels(r.dk.Name, k8slabel.ActiveGateComponentLabel)
	secretData := map[string][]byte{
		consts.TLSCrtDataName: pemCert,
		consts.TLSKeyDataName: pemPk,
		tlsCrtDataName:        pemCert,
	}

	secret, err := k8ssecret.Build(r.dk, r.dk.ActiveGate().GetTLSSecretName(), secretData, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		conditions.SetSecretGenFailed(r.dk.Conditions(), conditionType, err)

		return err
	}

	secret.Type = corev1.SecretTypeOpaque

	err = r.secrets.Create(ctx, secret)
	if err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	conditions.SetSecretCreated(r.dk.Conditions(), conditionType, secret.Name)

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
