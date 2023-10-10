package kubeobjects

import (
	"crypto/x509"
	"encoding/pem"
	"time"

	"github.com/go-logr/logr"
)

func ValidateCertificateExpiration(certData []byte, renewalThreshold time.Duration, now time.Time, log logr.Logger) (bool, error) {
	if block, _ := pem.Decode(certData); block == nil {
		log.Info("failed to parse certificate", "error", "can't decode PEM file")
		return false, nil
	} else if cert, err := x509.ParseCertificate(block.Bytes); err != nil {
		log.Info("failed to parse certificate", "error", err)
		return false, err
	} else if now.After(cert.NotAfter.Add(-renewalThreshold)) {
		log.Info("certificate is outdated, waiting for new ones", "Valid until", cert.NotAfter.UTC())
		return false, nil
	}
	return true, nil
}
