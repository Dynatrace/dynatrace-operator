// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"crypto/x509"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	tlsconsts "github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client       client.Client
	timeProvider *timeprovider.Provider
}

const (
	kubemonSelfSignedTLSCommonNameSuffix = "kubemon"
)

func NewReconciler(kubeClient client.Client) *Reconciler {
	return &Reconciler{
		client:       kubeClient,
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, _ = logd.NewFromContext(ctx, "gateway")

	if !dk.KubernetesMonitoring().IsEnabled() || !dk.KSPM().IsEnabled() {
		dk.KubernetesMonitoring().TLSSecretHash = ""

		if err := client.IgnoreNotFound(r.client.Delete(ctx, kubemonService(dk))); err != nil {
			return err
		}

		return client.IgnoreNotFound(r.client.Delete(ctx, tlsSecretSpec(dk, nil)))
	}

	if err := r.createService(ctx, dk); err != nil {
		return err
	}

	if dk.KubernetesMonitoring().TLSCertsRef == nil || dk.KubernetesMonitoring().TLSCertsRef.SecretName == "" {
		if err := r.createTLSSecret(ctx, dk); err != nil {
			return err
		}
	} else {
		// automatically created TLS secret is not used, delete it if exists
		_ = r.client.Delete(ctx, tlsSecretSpec(dk, nil))
	}

	return r.setStatusTLSSecretHash(ctx, dk)
}

func (r *Reconciler) createService(ctx context.Context, dk *dynakube.DynaKube) error {
	desired := kubemonService(dk)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}

	return k8sobject.RetryCreateOrUpdate(ctx, r.client, svc, func() error {
		svc.Labels = desired.Labels
		svc.Spec.Type = desired.Spec.Type
		svc.Spec.Selector = desired.Spec.Selector
		svc.Spec.Ports = desired.Spec.Ports

		return controllerutil.SetControllerReference(dk, svc, r.client.Scheme())
	})
}

func ServiceName(dynakubeName string) string {
	return dynakubeName + "-kubemon"
}

func kubemonService(dk *dynakube.DynaKube) *corev1.Service {
	ports := []corev1.ServicePort{
		{
			Name:       consts.HTTPSServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HTTPSServicePort,
			TargetPort: intstr.FromString(consts.HTTPSServicePortName),
		},
		{
			Name:       consts.HTTPServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HTTPServicePort,
			TargetPort: intstr.FromString(consts.HTTPServicePortName),
		},
	}

	labels := k8slabel.New(k8slabel.KubeMonComponentLabel, dk.Name, "")

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName(dk.Name),
			Namespace: dk.Namespace,
			Labels:    labels.AsMap(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels.AsSelector(),
			Ports:    ports,
		},
	}
}

func (r *Reconciler) createTLSSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	certificateData, err := r.tlsCertificateData(dk)
	if err != nil {
		return err
	}

	desired := tlsSecretSpec(dk, certificateData)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}

	return k8sobject.RetryCreateOrUpdate(ctx, r.client, secret, func() error {
		secret.Labels = desired.Labels
		secret.Type = desired.Type

		// desired key/certificate always have unique values so they are used only when
		// secret.Data is empty -> the secret doesn't exist yet
		// secret.Data is "broken"

		_, okTLSCrtDataName := secret.Data[tlsconsts.TLSCrtDataName]
		_, okTLSKeyDataName := secret.Data[tlsconsts.TLSKeyDataName]
		_, okTLSServerCrtDataName := secret.Data[tlsconsts.TLSServerCrtDataName]

		if !okTLSCrtDataName || !okTLSKeyDataName || !okTLSServerCrtDataName {
			secret.Data = desired.Data
		}

		return controllerutil.SetControllerReference(dk, secret, r.client.Scheme())
	})
}

func (r *Reconciler) tlsCertificateData(dk *dynakube.DynaKube) (map[string][]byte, error) {
	cert, err := certificates.New(r.timeProvider)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cert.Cert.DNSNames = certificates.AltNames(dk.Name, dk.Namespace, kubemonSelfSignedTLSCommonNameSuffix)
	cert.Cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
	cert.Cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	cert.Cert.Subject.CommonName = certificates.CommonName(dk.Name, dk.Namespace, kubemonSelfSignedTLSCommonNameSuffix)

	if err := cert.SelfSign(); err != nil {
		return nil, errors.WithStack(err)
	}

	pemCert, pemPk, err := cert.ToPEM()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return map[string][]byte{
		tlsconsts.TLSCrtDataName:       pemCert,
		tlsconsts.TLSKeyDataName:       pemPk,
		tlsconsts.TLSServerCrtDataName: pemCert,
	}, nil
}

func tlsSecretSpec(dk *dynakube.DynaKube, data map[string][]byte) *corev1.Secret {
	labels := k8slabel.New(k8slabel.KubeMonComponentLabel, dk.Name, "")

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetAutoTLSSecretName(),
			Namespace: dk.Namespace,
			Labels:    labels.AsMap(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
}

func (r *Reconciler) setStatusTLSSecretHash(ctx context.Context, dk *dynakube.DynaKube) error {
	var secret corev1.Secret

	// retry because a Secret created by the preceding createOrUpdate call may not be immediately visible in the API.
	err := retry.OnError(retry.DefaultBackoff, k8serrors.IsNotFound, func() error {
		err := r.client.Get(ctx, client.ObjectKey{Name: dk.KubernetesMonitoring().GetTLSSecretName(), Namespace: dk.Namespace}, &secret)
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if len(secret.Data) == 0 {
		dk.KubernetesMonitoring().TLSSecretHash = ""

		return nil
	}

	// custom secret may contain server.p12 field or tls.crt, tls.key fields
	hash, err := hasher.GenerateSecureHash(secret.Data)
	if err != nil {
		return errors.Wrap(err, "failed to hash TLS secret")
	}

	dk.KubernetesMonitoring().TLSSecretHash = hash

	return nil
}
