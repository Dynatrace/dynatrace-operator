package certificates

import "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"

func IsCertificateNeeded(dk *dynakube.DynaKube) bool {
	if dk.ActiveGate().IsApiEnabled() {
		if dk.ActiveGate().HasCaCert() {
			return true
		}
	} else if dk.Spec.TrustedCAs != "" {
		return true
	}

	return false
}
