package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
)

func (dk *DynaKube) ActiveGate() *activegate.ActiveGate {
	dk.Spec.ActiveGate.SetApiUrl(dk.ApiUrl())
	dk.Spec.ActiveGate.SetName(dk.Name)
	dk.Spec.ActiveGate.SetExtensionsDependency(dk.IsExtensionsEnabled())

	return &activegate.ActiveGate{
		Spec:   &dk.Spec.ActiveGate,
		Status: &dk.Status.ActiveGate,
	}
}
