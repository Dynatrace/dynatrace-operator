package v1

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("dynakube-conversion")

func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.DynaKube)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = src.Spec
	log.Info("Convert to called")
	if dst.Annotations == nil {
		dst.Annotations = map[string]string{}
	}
	dst.Annotations["CONVERT/TO"] = src.APIVersion

	return nil
}

func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.DynaKube)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = src.Spec
	log.Info("Convert from called")
	if dst.Annotations == nil {
		dst.Annotations = map[string]string{}
	}
	dst.Annotations["CONVERT/FROM"] = src.APIVersion

	return nil
}
