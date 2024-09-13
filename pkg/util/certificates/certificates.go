package certificates

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	intSerialNumberLimit  = 128
	defaultCertExpiration = 7 * 24 * time.Hour
	certificatePemHeader  = "CERTIFICATE"
	privateKeyPemHeader   = "PRIVATE KEY"
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
	Pk         *ecdsa.PrivateKey
	SignedCert []byte
	SignedPk   []byte
	signed     bool
}

func New() (*Certificate, error) {
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	cert := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               defaultCertSubject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(defaultCertExpiration),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	return &Certificate{Cert: cert, Pk: pk, signed: false}, nil
}

func (c *Certificate) SelfSign() error {
	certBytes, err := x509.CreateCertificate(rand.Reader, c.Cert, c.Cert, c.Pk.Public(), c.Pk)
	if err != nil {
		return err
	}

	pkBytes, err := x509.MarshalECPrivateKey(c.Pk)
	if err != nil {
		return err
	}

	c.SignedCert = certBytes
	c.SignedPk = pkBytes
	c.signed = true

	return nil
}

func (c *Certificate) CASign(ca *x509.Certificate, caPk *ecdsa.PrivateKey) error {
	certBytes, err := x509.CreateCertificate(rand.Reader, c.Cert, ca, c.Pk.Public(), caPk)
	if err != nil {
		return err
	}

	pkBytes, err := x509.MarshalECPrivateKey(c.Pk)
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
		Type:  certificatePemHeader,
		Bytes: c.SignedCert,
	})

	pemPk = pem.EncodeToMemory(&pem.Block{
		Type:  privateKeyPemHeader,
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
