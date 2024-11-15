package dynakube

func (dk *DynaKube) IsOtlpIngestEnabled() bool {
	return dk.Spec.EnableOtlpIngest
}
