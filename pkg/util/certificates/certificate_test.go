package certificates

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/stretchr/testify/require"
)

var randomTestData = []byte{
	45, 45, 45, 45, 45, 66,
	69, 71, 73, 78, 32, 67,
	69, 82, 84, 73, 70, 73,
	67, 65, 84, 69, 45, 45,
	45, 45, 45, 45, 45, 45,
	45, 45, 69, 78, 68, 32,
	67, 69, 82, 84, 73, 70,
	73, 67, 65, 84, 69, 45,
	45, 45, 45, 45,
}

var validateCertificateLog = logd.Get().WithName("test-certiciate-validation")

func TestValidateCertificateExpiration(t *testing.T) {
	t.Run("random data with resulting false validation", func(t *testing.T) {
		validate, err := ValidateCertificateExpiration(randomTestData, time.Minute, time.Now(), validateCertificateLog)
		require.NoError(t, err)
		require.False(t, validate)
	})
	t.Run("no data", func(t *testing.T) {
		validate, err := ValidateCertificateExpiration([]byte{}, time.Minute, time.Now(), validateCertificateLog)
		require.NoError(t, err)
		require.False(t, validate)
	})
}

func TestNewCertificate(t *testing.T) {
	t.Run("create new certificate", func(t *testing.T) {
		cert, err := New()

		require.NoError(t, err)

		require.NotNil(t, cert)
		require.NotNil(t, cert.Cert)
		require.NotNil(t, cert.Pk)
		require.Nil(t, cert.SignedCert)
		require.Nil(t, cert.SignedPk)
	})
}

func TestSelfSign(t *testing.T) {
	t.Run("self sign certificate", func(t *testing.T) {
		cert, _ := New()
		err := cert.SelfSign()

		require.NoError(t, err)

		require.NotNil(t, cert.SignedCert)
		require.NotNil(t, cert.SignedPk)
	})
}
