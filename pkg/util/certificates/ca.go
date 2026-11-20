// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package certificates

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"
)

const (
	// CACertDuration is the validity of the operator root CA.
	CACertDuration = 10 * 365 * 24 * time.Hour
	// ServerCertDuration is the default validity of certificates issued by the operator root CA.
	ServerCertDuration = 365 * 24 * time.Hour
	// ServerCertRenewalThreshold is the remaining validity below which a server certificate should be re-issued.
	ServerCertRenewalThreshold = 30 * 24 * time.Hour

	caCommonName = "Dynatrace Operator Root CA"
)

// ErrInvalidCA is returned (wrapped) when the provided CA material can not be used.
var ErrInvalidCA = errors.New("invalid CA")

// CA is a certificate authority that is able to issue server certificates.
type CA struct {
	Cert    *x509.Certificate
	Key     crypto.Signer
	CertPEM []byte
	KeyPEM  []byte
}

// NewCA generates a new self-signed root CA that is valid from now for CACertDuration.
func NewCA(now time.Time) (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA serial number: %w", err)
	}

	subject := defaultCertSubject
	subject.CommonName = caCommonName

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               subject,
		NotBefore:             now,
		NotAfter:              now.Add(CACertDuration),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated CA certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CA key: %w", err)
	}

	return &CA{
		Cert:    cert,
		Key:     key,
		CertPEM: pem.EncodeToMemory(&pem.Block{Type: pemHeaderCert, Bytes: der}),
		KeyPEM:  pem.EncodeToMemory(&pem.Block{Type: pemHeaderPk, Bytes: keyDER}),
	}, nil
}

// ParseCA validates and parses a CA from its PEM encoded certificate and key.
// Every failure wraps ErrInvalidCA; error messages never contain key material.
func ParseCA(certPEM, keyPEM []byte, now time.Time) (*CA, error) {
	if len(certPEM) == 0 {
		return nil, fmt.Errorf("%w: ca.crt is missing", ErrInvalidCA)
	}

	if len(keyPEM) == 0 {
		return nil, fmt.Errorf("%w: ca.key is missing", ErrInvalidCA)
	}

	// parses both PEMs and verifies that the key belongs to the certificate
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidCA, err.Error())
	}

	key, ok := pair.PrivateKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("%w: ca.key can not be used for signing", ErrInvalidCA)
	}

	cert := pair.Leaf

	if !cert.IsCA || !cert.BasicConstraintsValid || cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		return nil, fmt.Errorf("%w: certificate is not a CA", ErrInvalidCA)
	}

	if now.After(cert.NotAfter) {
		return nil, fmt.Errorf("%w: certificate expired at %s", ErrInvalidCA, cert.NotAfter.UTC().Format(time.RFC3339))
	}

	return &CA{Cert: cert, Key: key, CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

// ParseCertificatePEM parses the first PEM block of the given data as a certificate.
func ParseCertificatePEM(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != pemHeaderCert {
		return nil, errors.New("no PEM encoded certificate found")
	}

	return x509.ParseCertificate(block.Bytes)
}

// IssueServerCert issues a new server certificate (and key) for the given DNS names, signed by the CA.
// The validity is clamped to the validity of the CA.
func (ca *CA) IssueServerCert(dnsNames []string, now time.Time, validity time.Duration) ([]byte, []byte, error) {
	if len(dnsNames) == 0 {
		return nil, nil, errors.New("at least one DNS name is required")
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate server key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate server cert serial number: %w", err)
	}

	subject := defaultCertSubject
	subject.CommonName = dnsNames[0]

	notAfter := now.Add(validity)
	if notAfter.After(ca.Cert.NotAfter) {
		notAfter = ca.Cert.NotAfter
	}

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               subject,
		DNSNames:              dnsNames,
		NotBefore:             now,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca.Cert, key.Public(), ca.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create server certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal server key: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: pemHeaderCert, Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: pemHeaderPk, Bytes: keyDER}),
		nil
}
