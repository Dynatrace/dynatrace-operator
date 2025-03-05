package certificates

import "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"

func IsAGCertificateNeeded(dk *dynakube.DynaKube) bool {
	if dk.ActiveGate().IsEnabled() && dk.ActiveGate().IsApiEnabled() && dk.ActiveGate().HasCaCert() {
		return true
	}

	return false
}

func IsCACertificateNeeded(dk *dynakube.DynaKube) bool {
	if (!dk.ActiveGate().IsEnabled() || !dk.ActiveGate().IsApiEnabled()) && dk.Spec.TrustedCAs != "" {
		return true
	}

	return false
}
