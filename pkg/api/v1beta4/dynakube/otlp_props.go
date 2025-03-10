package dynakube

func (dk *DynaKube) IsOTLPingestEnabled() bool {
	return dk.Spec.EnableOTLPingest
}
