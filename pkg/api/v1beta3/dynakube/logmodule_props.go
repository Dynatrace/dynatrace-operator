package dynakube

func (dk *DynaKube) NeedsLogModule() bool {
	return dk.Spec.LogModule.Enabled
}
