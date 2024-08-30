package certificates

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	// "golang.org/x/crypto/pkcs12"

	"software.sslmate.com/src/go-pkcs12"
)

const (
	intSerialNumberLimit                  = 128
	certPassword                          = "changeit"
	SelfSignedCertificateSecretName       = "%s-activegate-selfsigned-cert"
	SelfSignedCertificateActiveGateDomain = "%s-activegate.%s"

	certificatePemHeader = "CERTIFICATE"
	privateKeyPemHeader  = "PRIVATE KEY"

	TlsCrtDataMapKey    = "tls.crt"
	TlsKeyDataMapKey    = "tls.key"
	ServerCrtDataMapKey = "server.crt"
	ServerP12DataMapKey = "server.p12"
	PasswordDataMapKey  = "password"
)

var serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), intSerialNumberLimit)

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

func CreateSelfSignedCertificate(domain string, altNames []string, ip string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number for tls certificate: %w", err)
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	//extSubjectAltName := pkix.Extension{}
	//extSubjectAltName.Id = asn1.ObjectIdentifier{2, 5, 29, 17}
	//extSubjectAltName.Critical = false
	//extSubjectAltName.Value = []byte(strings.Join(altNames[:], ", "))
	//var e error
	//extSubjectAltName.Value, e = asn1.Marshal(altNames[:])
	//if e != nil {
	//	return nil, nil, err
	//}

	netIp := net.ParseIP(ip)

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{"AT"},
			Province:           []string{"UA"},
			Locality:           []string{"Linz"},
			Organization:       []string{"Dynatrace"},
			OrganizationalUnit: []string{"Operator Self-Signed"},
			CommonName:         domain,
		},
		// ExtraExtensions: []pkix.Extension{extSubjectAltName},
		//Extensions: []pkix.Extension{extSubjectAltName},

		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(7 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		IsCA: true,

		DNSNames:    altNames,
		IPAddresses: []net.IP{netIp},
	}

	return cert, privateKey, nil
}

func CreateP12CertificateSecretData(cert *x509.Certificate, privateKey *ecdsa.PrivateKey) (map[string][]byte, error) {
	data := map[string][]byte{}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, privateKey.Public(), privateKey)
	if err != nil {
		return nil, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: certificatePemHeader, Bytes: certBytes})
	p12Data, err := pkcs12.Modern.Encode(privateKey, cert, []*x509.Certificate{cert}, certPassword)

	data[ServerCrtDataMapKey] = certPem
	data[ServerP12DataMapKey] = p12Data
	data[PasswordDataMapKey] = []byte(certPassword)

	return data, nil
}
