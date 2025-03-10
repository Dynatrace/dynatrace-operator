package dynakube

func (dk *DynaKube) OtelCollectorStatefulsetName() string {
	return dk.Name + "-extensions-collector"
}

func (dk *DynaKube) IsAGCertificateNeeded() bool {
	if dk.isInClusterActiveGate() && dk.ActiveGate().HasCaCert() {
		return true
	}

	return false
}
func (dk *DynaKube) IsCACertificateNeeded() bool {
	if !dk.isInClusterActiveGate() && dk.Spec.TrustedCAs != "" {
		return true
	}

	return false
}
func (dk *DynaKube) isInClusterActiveGate() bool {
	if dk.ActiveGate().IsEnabled() && dk.ActiveGate().IsApiEnabled() {
		return true
	}

	return false
}
