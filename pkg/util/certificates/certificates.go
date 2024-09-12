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
	"net"
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
	pk         *ecdsa.PrivateKey
	signedCert []byte
	signedPk   []byte
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

	return &Certificate{Cert: cert, pk: pk, signed: false}, nil
}

func New2(domain string, altNames []string, ip string, keyUsages []x509.ExtKeyUsage) (*Certificate, error) {
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
		DNSNames:              altNames,
		ExtKeyUsage:           keyUsages,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(defaultCertExpiration),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	if domain != "" {
		cert.Subject.CommonName = domain
	}

	if ip != "" {
		netIp := net.ParseIP(ip)
		cert.IPAddresses = []net.IP{netIp}
	}

	return &Certificate{Cert: cert, pk: pk, signed: false}, nil
}

func (c *Certificate) SelfSign() error {
	certBytes, err := x509.CreateCertificate(rand.Reader, c.Cert, c.Cert, c.pk.Public(), c.pk)
	if err != nil {
		return err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: certificatePemHeader, Bytes: certBytes})

	pkx509Encoded, err := x509.MarshalECPrivateKey(c.pk)
	if err != nil {
		return err
	}

	pkPem := pem.EncodeToMemory(&pem.Block{
		Type:  privateKeyPemHeader,
		Bytes: pkx509Encoded,
	})

	c.signedCert = certPem
	c.signedPk = pkPem
	c.signed = true

	return nil
}

func (c *Certificate) CASign(ca *x509.Certificate, caPk *ecdsa.PrivateKey) error {
	certBytes, err := x509.CreateCertificate(rand.Reader, c.Cert, ca, c.pk.Public(), caPk)
	if err != nil {
		return err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: certificatePemHeader, Bytes: certBytes})

	pkx509Encoded, err := x509.MarshalECPrivateKey(c.pk)
	if err != nil {
		return err
	}

	pkPem := pem.EncodeToMemory(&pem.Block{
		Type:  privateKeyPemHeader,
		Bytes: pkx509Encoded,
	})

	c.signedCert = certPem
	c.signedPk = pkPem
	c.signed = true

	return nil
}

func (c *Certificate) ToPEM() (certPem []byte, pkPem []byte, err error) {
	if !c.signed {
		return nil, nil, errors.New("failed parsing certificate to PEM format: certificate hasn't been signed")
	}

	certPem = pem.EncodeToMemory(&pem.Block{
		Type:  certificatePemHeader,
		Bytes: c.signedCert,
	})

	pkPem = pem.EncodeToMemory(&pem.Block{
		Type:  privateKeyPemHeader,
		Bytes: c.signedPk,
	})

	return certPem, pkPem, nil
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
