package v1

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("dynakube-conversion")

func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	log.Info("Convert to called")
	dst := dstRaw.(*v1alpha1.DynaKube)
	dst.ObjectMeta = src.ObjectMeta
	//dst.Spec = src.Spec
	dst.Status = src.Status
	return nil
}

func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	log.Info("Convert from from")
	src := srcRaw.(*v1alpha1.DynaKube)
	dst.ObjectMeta = src.ObjectMeta
	//dst.Spec = src.Spec
	dst.Status = src.Status
	return nil
}
