package certificates

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	intSerialNumberLimit = 128

	certificatePemHeader = "CERTIFICATE"
	privateKeyPemHeader  = "PRIVATE KEY"
)

var serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), intSerialNumberLimit)

type Certificate struct {
	cert       *x509.Certificate
	pk         *ecdsa.PrivateKey
	signedCert []byte
	signedPk   []byte
}

func New(domain string, altNames []string, ip string) (*Certificate, error) {
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	netIp := net.ParseIP(ip)

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{"AT"},
			Province:           []string{"UA"},
			Locality:           []string{"Linz"},
			Organization:       []string{"Dynatrace"},
			OrganizationalUnit: []string{"Operator"},
			CommonName:         domain,
		},
		DNSNames:    altNames,
		IPAddresses: []net.IP{netIp},

		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(7 * 24 * time.Hour),

		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	return &Certificate{cert: cert, pk: pk}, nil
}

func (c *Certificate) SelfSign() error {
	certBytes, err := x509.CreateCertificate(rand.Reader, c.cert, c.cert, c.pk.Public(), c.pk)
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

	return nil
}

func (c *Certificate) CaSign(ca *x509.Certificate, caPk *ecdsa.PrivateKey) error {
	certBytes, err := x509.CreateCertificate(rand.Reader, c.cert, ca, c.pk.Public(), caPk)
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

	return nil
}

func (c *Certificate) ToPEM() (certPem []byte, pkPem []byte) {
	certPem = pem.EncodeToMemory(&pem.Block{
		Type:  certificatePemHeader,
		Bytes: c.signedCert,
	})

	pkPem = pem.EncodeToMemory(&pem.Block{
		Type:  privateKeyPemHeader,
		Bytes: c.signedPk,
	})

	return certPem, pkPem
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
