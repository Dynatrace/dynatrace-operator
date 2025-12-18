package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/consts"

func (dk *DynaKube) OtelCollectorStatefulsetName() string {
	return dk.Name + consts.OTELCollectorNameSuffix
}

func (dk *DynaKube) IsAGCertificateNeeded() bool {
	if dk.ActiveGate().IsEnabled() && dk.ActiveGate().HasCaCert() {
		return true
	}

	return false
}

func (dk *DynaKube) IsCACertificateNeeded() bool {
	if !dk.ActiveGate().IsEnabled() && dk.Spec.TrustedCAs != "" {
		return true
	}

	return false
}
