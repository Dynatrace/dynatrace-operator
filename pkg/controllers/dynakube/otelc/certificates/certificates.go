package certificates

import "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"

func IsAGCertificateNeeded(dk *dynakube.DynaKube) bool {
	if isInClusterActiveGate(dk) {
		if dk.ActiveGate().HasCaCert() {
			return true
		}
	}

	return false
}

func IsCACertificateNeeded(dk *dynakube.DynaKube) bool {
	if !isInClusterActiveGate(dk) {
		if dk.Spec.TrustedCAs != "" {
			return true
		}
	}

	return false
}

func isInClusterActiveGate(dk *dynakube.DynaKube) bool {
	if dk.ActiveGate().IsEnabled() && dk.ActiveGate().IsApiEnabled() {
		return true
	}

	return false
}
