package dynakube

func (dk *DynaKube) IsKSPMEnabled() bool {
	return dk.Spec.Kspm.Enabled
}
