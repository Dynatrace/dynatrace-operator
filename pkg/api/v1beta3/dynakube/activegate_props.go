package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
)

func (dk *DynaKube) ActiveGate() *activegate.ActiveGate {
	dk.Spec.ActiveGate.SetApiUrl(dk.ApiUrl())
	dk.Spec.ActiveGate.SetName(dk.Name)
	dk.Spec.ActiveGate.SetTrustedCAs(dk.Spec.TrustedCAs)
	dk.Spec.ActiveGate.SetExtensionsDependency(dk.IsExtensionsEnabled())
	dk.Spec.ActiveGate.SetOTLPingestDependency(dk.IsOTLPingestEnabled())

	return &activegate.ActiveGate{
		Spec:   &dk.Spec.ActiveGate,
		Status: &dk.Status.ActiveGate,
	}
}
