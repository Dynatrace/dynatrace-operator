package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"

func (dk *DynaKube) MetadataEnrichment() *metadataenrichment.MetadataEnrichment {
	m := &metadataenrichment.MetadataEnrichment{
		Spec: &dk.Spec.MetadataEnrichment,
	}

	return m
}
