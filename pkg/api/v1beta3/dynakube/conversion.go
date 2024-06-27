package dynakube

import (
	v1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertFrom converts from the Hub version (v1beta2) to this version (v1beta3).
func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	_ = srcRaw.(*v1beta2.DynaKube)
	return nil
}

// ConvertTo converts this v1beta3.DynaKube to the Hub version (v1beta2.DynaKube).
func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	_ = dstRaw.(*v1beta2.DynaKube)
	// TODO marshal as anotationkeyy
	return nil
}
