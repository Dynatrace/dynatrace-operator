// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// CertHashAnnotation holds the hash of tls.crt and ca.crt of a server cert Secret,
	// consumers can put it into their pod template annotations without hashing the Secret themselves.
	CertHashAnnotation = api.InternalFlagPrefix + "tls-cert-hash"
)

// CAUnavailableError is returned if the operator root CA does not exist (yet) or is invalid.
// Callers are expected to requeue and to set their own condition.
type CAUnavailableError struct {
	Err error
}

func (e *CAUnavailableError) Error() string {
	return "operator root CA unavailable: " + e.Err.Error()
}

func (e *CAUnavailableError) Unwrap() error {
	return e.Err
}

func IsCAUnavailable(err error) bool {
	var target *CAUnavailableError

	return errors.As(err, &target)
}

// serviceDNSNames returns the DNS names under which a Service is reachable in the cluster.
func serviceDNSNames(serviceName, namespace string) []string {
	return []string{
		serviceName + "." + namespace, // kept as first, to remain compatible with previous impl
		serviceName,
		serviceName + "." + namespace + ".svc",
		serviceName + "." + namespace + ".svc.cluster.local",
	}
}

// Issuer manages the server cert Secret of a component, that is signed by the operator root CA.
type Issuer struct {
	client            client.Client
	operatorNamespace string
	// componentName is used for labeling the Secret
	componentName string
	// serviceName is the name of the Service of the component, the server certificate is valid for its DNS names
	serviceName string
}

func NewIssuer(clt client.Client, operatorNamespace, componentName, serviceName string) *Issuer {
	return &Issuer{
		client:            clt,
		operatorNamespace: operatorNamespace,
		componentName:     componentName,
		serviceName:       serviceName,
	}
}

// EnsureServerCertSecret makes sure that a TLS Secret named secretName, owned by owner, exists in the
// namespace of the owner, and contains a valid server certificate for the DNS names of the service of the Issuer
// (in the namespace of the owner), signed by the current operator root CA.
// The certificate is re-issued if the service or the CA changed, or if it is about to expire.
// It returns the hash of tls.crt and ca.crt that is also stored in the CertHashAnnotation of the Secret.
func (i *Issuer) EnsureServerCertSecret(ctx context.Context, owner client.Object, secretName string) (string, error) {
	if owner == nil || owner.GetNamespace() == "" {
		return "", errors.New("owner with a namespace is required to ensure a server certificate secret")
	}

	if i.serviceName == "" {
		return "", errors.New("service name is required to ensure a server certificate secret")
	}

	dnsNames := serviceDNSNames(i.serviceName, owner.GetNamespace())

	now := time.Now()

	ca, err := i.loadCA(ctx, now)
	if err != nil {
		return "", err
	}

	var (
		hash     string
		reissued bool
	)

	secret := &corev1.Secret{Name: secretName, Namespace: owner.GetNamespace()}

	err = k8sobject.RetryCreateOrUpdate(ctx, i.client, secret, func() error {
		var err error

		k8slabel.New(i.componentName, owner.GetName(), "").MergeInto(secret)

		err = controllerutil.SetControllerReference(owner, secret, i.client.Scheme())
		if err != nil {
			return err
		}

		hash, reissued, err = i.mutate(secret, ca, dnsNames, now)

		return err
	})
	if err != nil {
		return "", err
	}

	if reissued {
		logd.FromContext(ctx).Info("issued server certificate", "secret", secret.Name, "namespace", secret.Namespace)
	}

	return hash, nil
}

func (i *Issuer) loadCA(ctx context.Context, now time.Time) (*certificates.CA, error) {
	ca, err := loadCA(ctx, i.client, i.operatorNamespace, now)
	if err != nil {
		if errors.Is(err, ErrCANotFound) || errors.Is(err, certificates.ErrInvalidCA) {
			return nil, &CAUnavailableError{Err: err}
		}

		return nil, fmt.Errorf("failed to load operator root CA: %w", err)
	}

	return ca, nil
}

// mutate brings the secret to the desired state, the certificate is only re-issued if needed, so a valid secret stays unchanged.
func (i *Issuer) mutate(secret *corev1.Secret, ca *certificates.CA, dnsNames []string, now time.Time) (hash string, reissued bool, err error) {
	// the type is immutable
	if secret.ResourceVersion != "" && secret.Type != corev1.SecretTypeTLS {
		return "", false, k8sobject.ErrRecreate
	}

	secret.Type = corev1.SecretTypeTLS

	want := slices.Compact(slices.Sorted(slices.Values(dnsNames)))
	if needsReissue(secret, ca, want, now) {
		certPEM, keyPEM, err := ca.IssueServerCert(dnsNames, now, certificates.ServerCertDuration)
		if err != nil {
			return "", false, err
		}

		chain := slices.Concat(certPEM, ca.CertPEM)

		secret.Data = map[string][]byte{
			corev1.TLSCertKey:       chain,
			corev1.TLSPrivateKeyKey: keyPEM,
			CACertKey:               ca.CertPEM,
		}

		reissued = true
	}

	hash, err = computeHash(secret.Data)
	if err != nil {
		return "", false, err
	}

	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}

	secret.Annotations[CertHashAnnotation] = hash

	return hash, reissued, nil
}

func needsReissue(existing *corev1.Secret, ca *certificates.CA, wantDNSNames []string, now time.Time) bool {
	if !bytes.Equal(existing.Data[CACertKey], ca.CertPEM) {
		return true
	}

	if len(existing.Data[corev1.TLSPrivateKeyKey]) == 0 {
		return true
	}

	leaf, err := certificates.ParseCertificatePEM(existing.Data[corev1.TLSCertKey])
	if err != nil {
		return true
	}

	if !slices.Equal(slices.Sorted(slices.Values(leaf.DNSNames)), wantDNSNames) {
		return true
	}

	// a leaf that is clamped to the CA validity can not be renewed any longer than it is
	return now.Add(certificates.ServerCertRenewalThreshold).After(leaf.NotAfter) && leaf.NotAfter.Before(ca.Cert.NotAfter)
}

// computeHash computes the hash of the tls.crt and ca.crt entries of the given Secret data.
func computeHash(data map[string][]byte) (string, error) {
	return hasher.GenerateSecureHash(map[string]string{
		corev1.TLSCertKey: string(data[corev1.TLSCertKey]),
		CACertKey:         string(data[CACertKey]),
	})
}
