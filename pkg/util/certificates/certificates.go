package certificates

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
)

const (
	intSerialNumberLimit  = 128
	defaultCertExpiration = 100 * 365 * 24 * time.Hour // 100 years = no expiration
	rsaKeyBits            = 4096
	pemHeaderCert         = "CERTIFICATE"
	pemHeaderPk           = "PRIVATE KEY"
)

var (
	serialNumberLimit  = new(big.Int).Lsh(big.NewInt(1), intSerialNumberLimit)
	defaultCertSubject = pkix.Name{
		Country:            []string{"AT"},
		Province:           []string{"UA"},
		Locality:           []string{"Linz"},
		Organization:       []string{"Dynatrace"},
		OrganizationalUnit: []string{"Operator"},
	}
)

type Certificate struct {
	Cert       *x509.Certificate
	Pk         *rsa.PrivateKey
	SignedCert []byte
	SignedPk   []byte
	signed     bool
}

func New(timeProvider *timeprovider.Provider) (*Certificate, error) {
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	pk, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, err
	}

	cert := &x509.Certificate{
		SerialNumber:       serialNumber,
		Subject:            defaultCertSubject,
		NotBefore:          timeProvider.Now().Time,
		NotAfter:           timeProvider.Now().Time.Add(defaultCertExpiration),
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	return &Certificate{Cert: cert, Pk: pk, signed: false}, nil
}

func (c *Certificate) SelfSign() error {
	certBytes, err := x509.CreateCertificate(rand.Reader, c.Cert, c.Cert, c.Pk.Public(), c.Pk)
	if err != nil {
		return err
	}

	pkBytes, err := x509.MarshalPKCS8PrivateKey(c.Pk)
	if err != nil {
		return err
	}

	c.SignedCert = certBytes
	c.SignedPk = pkBytes
	c.signed = true

	return nil
}

func (c *Certificate) ToPEM() (pemCert []byte, pemPk []byte, err error) {
	if !c.signed {
		return nil, nil, errors.New("failed parsing certificate to PEM format: certificate hasn't been signed")
	}

	pemCert = pem.EncodeToMemory(&pem.Block{
		Type:  pemHeaderCert,
		Bytes: c.SignedCert,
	})

	pemPk = pem.EncodeToMemory(&pem.Block{
		Type:  pemHeaderPk,
		Bytes: c.SignedPk,
	})

	return pemCert, pemPk, nil
}

func ValidateCertificateExpiration(certData []byte, renewalThreshold time.Duration, now time.Time, log logd.Logger) (bool, error) {
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
