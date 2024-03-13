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
