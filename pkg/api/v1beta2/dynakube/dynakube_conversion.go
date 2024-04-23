package dynakube

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts v1beta2 to v1beta1.
func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	return nil
}

// ConvertFrom converts v1beta1 to v1beta2.
func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	return nil
}
