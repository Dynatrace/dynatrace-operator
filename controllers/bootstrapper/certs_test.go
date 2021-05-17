package bootstrapper

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestCertsValidation(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout))

	now, _ := time.Parse(time.RFC3339, "2018-01-10T00:00:00Z")
	domain := "dynatrace-oneagent-webhook.webhook.svc"
	firstCerts := Certs{Log: logger, Domain: domain, now: now}

	require.NoError(t, firstCerts.ValidateCerts())
	require.Equal(t, len(firstCerts.Data), 4)
	requireValidCerts(t, domain, now.Add(5*time.Minute), firstCerts.Data["ca.crt"], firstCerts.Data["tls.crt"])

	t.Run("up-to-date certs", func(t *testing.T) {
		newTime := now.Add(5 * time.Minute)

		newCerts := Certs{Log: logger, Domain: domain, SrcData: firstCerts.Data, now: newTime}
		require.NoError(t, newCerts.ValidateCerts())
		requireValidCerts(t, domain, newTime, newCerts.Data["ca.crt"], newCerts.Data["tls.crt"])

		// No changes should have been applied.
		assert.Equal(t, string(firstCerts.Data["ca.crt"]), string(newCerts.Data["ca.crt"]))
		assert.Equal(t, string(firstCerts.Data["ca.key"]), string(newCerts.Data["ca.key"]))
		assert.Equal(t, string(firstCerts.Data["tls.crt"]), string(newCerts.Data["tls.crt"]))
		assert.Equal(t, string(firstCerts.Data["tls.key"]), string(newCerts.Data["tls.key"]))
	})

	t.Run("outdated server certs", func(t *testing.T) {
		newTime := now.Add((6*24 + 22) * time.Hour) // 6d22h

		newCerts := Certs{Log: logger, Domain: domain, SrcData: firstCerts.Data, now: newTime}
		require.NoError(t, newCerts.ValidateCerts())
		requireValidCerts(t, domain, newTime, newCerts.Data["ca.crt"], newCerts.Data["tls.crt"])

		// Server certificates should have been updated.
		assert.Equal(t, string(firstCerts.Data["ca.crt"]), string(newCerts.Data["ca.crt"]))
		assert.Equal(t, string(firstCerts.Data["ca.key"]), string(newCerts.Data["ca.key"]))
		assert.NotEqual(t, string(firstCerts.Data["tls.crt"]), string(newCerts.Data["tls.crt"]))
		assert.NotEqual(t, string(firstCerts.Data["tls.key"]), string(newCerts.Data["tls.key"]))
	})

	t.Run("outdated root certs", func(t *testing.T) {
		newTime := now.Add((364*24 + 22) * time.Hour) // 364d22h

		newCerts := Certs{Log: logger, Domain: domain, SrcData: firstCerts.Data, now: newTime}
		require.NoError(t, newCerts.ValidateCerts())
		requireValidCerts(t, domain, newTime, newCerts.Data["ca.crt"], newCerts.Data["tls.crt"])

		// Server certificates should have been updated.
		assert.NotEqual(t, string(firstCerts.Data["ca.crt"]), string(newCerts.Data["ca.crt"]))
		assert.NotEqual(t, string(firstCerts.Data["ca.key"]), string(newCerts.Data["ca.key"]))
		assert.NotEqual(t, string(firstCerts.Data["tls.crt"]), string(newCerts.Data["tls.crt"]))
		assert.NotEqual(t, string(firstCerts.Data["tls.key"]), string(newCerts.Data["tls.key"]))
	})
}

func requireValidCerts(t *testing.T, domain string, now time.Time, caCert, tlsCert []byte) {
	caCerts := x509.NewCertPool()
	require.True(t, caCerts.AppendCertsFromPEM(caCert))

	block, _ := pem.Decode(tlsCert)
	require.NotNil(t, block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	_, err = cert.Verify(x509.VerifyOptions{DNSName: domain, CurrentTime: now, Roots: caCerts})
	require.NoError(t, err)
}
