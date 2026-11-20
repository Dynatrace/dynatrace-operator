// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package certificates

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestNewCAAndParseCA(t *testing.T) {
	ca, err := NewCA(testNow)
	require.NoError(t, err)
	assert.True(t, ca.Cert.IsCA)
	assert.Equal(t, caCommonName, ca.Cert.Subject.CommonName)
	assert.Empty(t, ca.Cert.DNSNames)
	assert.Empty(t, ca.Cert.ExtKeyUsage)
	assert.Equal(t, testNow.Add(CACertDuration).Unix(), ca.Cert.NotAfter.Unix())

	parsed, err := ParseCA(ca.CertPEM, ca.KeyPEM, testNow)
	require.NoError(t, err)
	assert.True(t, parsed.Cert.Equal(ca.Cert))
	assert.Equal(t, ca.CertPEM, parsed.CertPEM)
}

func TestParseCAInvalid(t *testing.T) {
	ca, err := NewCA(testNow)
	require.NoError(t, err)

	other, err := NewCA(testNow)
	require.NoError(t, err)

	leafCert, leafKey, err := ca.IssueServerCert([]string{"a.example.com"}, testNow, time.Hour)
	require.NoError(t, err)

	tests := []struct {
		name    string
		cert    []byte
		key     []byte
		now     time.Time
		message string
	}{
		{"missing cert", nil, ca.KeyPEM, testNow, "ca.crt is missing"},
		{"missing key", ca.CertPEM, nil, testNow, "ca.key is missing"},
		{"garbage cert", []byte("garbage"), ca.KeyPEM, testNow, ""},
		{"garbage key", ca.CertPEM, []byte("garbage"), testNow, "failed to find any PEM data in key input"},
		{"unparsable key", ca.CertPEM, pem.EncodeToMemory(&pem.Block{Type: pemHeaderPk, Bytes: []byte("x")}), testNow, "failed to parse private key"},
		{"not a CA", leafCert, leafKey, testNow, "not a CA"},
		{"foreign key", ca.CertPEM, other.KeyPEM, testNow, "does not match"},
		{"expired", ca.CertPEM, ca.KeyPEM, testNow.Add(CACertDuration + time.Hour), "expired"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseCA(test.cert, test.key, test.now)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidCA)
			assert.Contains(t, err.Error(), test.message)
		})
	}
}

func TestParseCALegacyECKey(t *testing.T) {
	ca, err := NewCA(testNow)
	require.NoError(t, err)

	ecKey, ok := ca.Key.(*ecdsa.PrivateKey)
	require.True(t, ok)

	der, err := x509.MarshalECPrivateKey(ecKey)
	require.NoError(t, err)

	sec1 := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})

	parsed, err := ParseCA(ca.CertPEM, sec1, testNow)
	require.NoError(t, err)
	assert.True(t, parsed.Cert.Equal(ca.Cert))
}

func TestIssueServerCert(t *testing.T) {
	ca, err := NewCA(testNow)
	require.NoError(t, err)

	names := []string{"svc", "svc.ns", "svc.ns.svc", "svc.ns.svc.cluster.local"}

	certPEM, keyPEM, err := ca.IssueServerCert(names, testNow, ServerCertDuration)
	require.NoError(t, err)
	assert.NotEmpty(t, keyPEM)

	leaf, err := ParseCertificatePEM(certPEM)
	require.NoError(t, err)
	assert.False(t, leaf.IsCA)
	assert.Equal(t, "svc", leaf.Subject.CommonName)

	roots := x509.NewCertPool()
	roots.AddCert(ca.Cert)

	for _, name := range names {
		_, err := leaf.Verify(x509.VerifyOptions{
			DNSName:     name,
			Roots:       roots,
			CurrentTime: testNow.Add(time.Hour),
			KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		})
		require.NoError(t, err, name)
	}

	_, err = leaf.Verify(x509.VerifyOptions{
		DNSName:     "other.example.com",
		Roots:       roots,
		CurrentTime: testNow.Add(time.Hour),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	var hostErr x509.HostnameError
	require.ErrorAs(t, err, &hostErr)

	t.Run("no names", func(t *testing.T) {
		_, _, err := ca.IssueServerCert(nil, testNow, ServerCertDuration)
		require.Error(t, err)
	})
	t.Run("clamped to CA validity", func(t *testing.T) {
		late := testNow.Add(CACertDuration - time.Hour)
		certPEM, _, err := ca.IssueServerCert(names, late, ServerCertDuration)
		require.NoError(t, err)

		leaf, err := ParseCertificatePEM(certPEM)
		require.NoError(t, err)
		assert.True(t, leaf.NotAfter.Equal(ca.Cert.NotAfter))
	})
}

func TestParseCertificatePEM(t *testing.T) {
	_, err := ParseCertificatePEM([]byte("garbage"))
	require.Error(t, err)

	_, err = ParseCertificatePEM(pem.EncodeToMemory(&pem.Block{Type: pemHeaderCert, Bytes: []byte("x")}))
	require.Error(t, err)
}
