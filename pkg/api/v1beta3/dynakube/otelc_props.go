package dynakube

func (dk *DynaKube) OtelCollectorStatefulsetName() string {
	return dk.Name + "-extensions-collector"
}
