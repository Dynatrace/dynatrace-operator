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
	"time"
)

var serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), 128)

const (
	renewalThreshold = 12 * time.Hour

	RootKey     = "ca.key"
	RootCert    = "ca.crt"
	RootCertOld = "ca.crt.old"
	ServerKey   = "tls.key"
	ServerCert  = "tls.crt"
)

// Certs handles creation and renewal of CA and SSL/TLS server certificates.
type Certs struct {
	Domain  string
	SrcData map[string][]byte
	Data    map[string][]byte

	Now time.Time

	rootPrivateKey *ecdsa.PrivateKey
	rootPublicCert *x509.Certificate
}

// ValidateCerts checks for certificates and keys on cs.SrcData and renews them if needed. The existing (or new)
// certificates will be stored on cs.Data.
func (cs *Certs) ValidateCerts() error {
	cs.Data = map[string][]byte{}
	if cs.SrcData != nil {
		for k, v := range cs.SrcData {
			cs.Data[k] = v
		}
	}

	now := time.Now().UTC()
	if !cs.Now.IsZero() {
		now = cs.Now
	}

	renewRootCerts := cs.validateRootCerts(now)
	if renewRootCerts {
		if err := cs.generateRootCerts(cs.Domain, now); err != nil {
			return err
		}
	}

	if renewRootCerts || cs.validateServerCerts(now) {
		return cs.generateServerCerts(cs.Domain, now)
	}

	return nil
}

func (cs *Certs) validateRootCerts(now time.Time) bool {
	if cs.Data[RootKey] == nil || cs.Data[RootCert] == nil {
		log.Info("no root certificates found, creating")
		return true
	}

	var err error

	if block, _ := pem.Decode(cs.Data[RootCert]); block == nil {
		log.Info("failed to parse root certificates, renewing", "error", "can't decode PEM file")
		return true
	} else if cs.rootPublicCert, err = x509.ParseCertificate(block.Bytes); err != nil {
		log.Info("failed to parse root certificates, renewing", "error", err)
		return true
	} else if now.After(cs.rootPublicCert.NotAfter.Add(-renewalThreshold)) {
		log.Info("root certificates are about to expire, renewing", "current", now, "expiration", cs.rootPublicCert.NotAfter)
		return true
	}

	if block, _ := pem.Decode(cs.Data[RootKey]); block == nil {
		log.Info("failed to parse root key, renewing", "error", "can't decode PEM file")
		return true
	} else if cs.rootPrivateKey, err = x509.ParseECPrivateKey(block.Bytes); err != nil {
		log.Info("failed to parse root key, renewing", "error", err)
		return true
	}

	return false
}

func (cs *Certs) validateServerCerts(now time.Time) bool {
	if cs.Data[ServerKey] == nil || cs.Data[ServerCert] == nil {
		log.Info("no server certificates found, creating")
		return true
	}

	block, _ := pem.Decode(cs.Data[ServerCert])
	if block == nil {
		log.Info("failed to parse server certificates, renewing", "error", "can't decode PEM file")
		return true
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Info("failed to parse server certificates, renewing", "error", err)
		return true
	}

	if now.After(cert.NotAfter.Add(-renewalThreshold)) {
		log.Info("server certificates are about to expire, renewing", "current", now, "expiration", cert.NotAfter)
		return true
	}

	return false
}

func (cs *Certs) generateRootCerts(domain string, now time.Time) error {
	// Generate CA root keys
	log.Info("generating root certificates")
	privateKey, err := cs.generatePrivateKey(RootKey)
	if err != nil {
		return err
	}
	cs.rootPrivateKey = privateKey

	// Generate CA root certificates
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number for root certificates: %w", err)
	}

	cs.rootPublicCert = &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{"AT"},
			Province:           []string{"UA"},
			Locality:           []string{"Linz"},
			Organization:       []string{"Dynatrace"},
			OrganizationalUnit: []string{"OneAgent Webhook"},
			CommonName:         domain,
		},
		IsCA: true,

		NotBefore: now,
		NotAfter:  now.Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	rootPublicCertDER, err := x509.CreateCertificate(
		rand.Reader,
		cs.rootPublicCert,
		cs.rootPublicCert,
		cs.rootPrivateKey.Public(),
		cs.rootPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to generate root certificates: %w", err)
	}

	cs.Data[RootCertOld] = cs.Data[RootCert]
	cs.Data[RootCert] = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootPublicCertDER})

	log.Info("root certificates generated")
	return nil
}

func (cs *Certs) generateServerCerts(domain string, now time.Time) error {
	// Generate server keys
	log.Info("generating server certificates")
	privateKey, err := cs.generatePrivateKey(ServerKey)
	if err != nil {
		return err
	}

	// Generate server certificates
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number for server certificates: %w", err)
	}

	tpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{"AT"},
			Province:           []string{"UA"},
			Locality:           []string{"Linz"},
			Organization:       []string{"Dynatrace"},
			OrganizationalUnit: []string{"OneAgent Webhook"},
			CommonName:         domain,
		},

		DNSNames: []string{domain},

		NotBefore: now,
		NotAfter:  now.Add(7 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	serverPublicCertDER, err := x509.CreateCertificate(rand.Reader, tpl, cs.rootPublicCert, privateKey.Public(), cs.rootPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to generate server certificates: %w", err)
	}

	cs.Data[ServerCert] = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverPublicCertDER})
	log.Info("server certificates generated")
	return nil
}

func (cs *Certs) generatePrivateKey(dataKey string) (*ecdsa.PrivateKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server private key: %w", err)
	}

	x509Encoded, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	cs.Data[dataKey] = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509Encoded,
	})
	return privateKey, nil
}
