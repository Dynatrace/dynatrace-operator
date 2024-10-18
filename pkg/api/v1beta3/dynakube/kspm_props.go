package dynakube

const (
	KSPMSecretKey = "kspm-token"
)

func (dk *DynaKube) IsKSPMEnabled() bool {
	return dk.Spec.Kspm.Enabled
}

func (dk *DynaKube) GetKSPMSecretName() string {
	return dk.Name + "-" + KSPMSecretKey
}
