package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
)

func (dk *DynaKube) ActiveGate() *activegate.ActiveGate {
	dk.Spec.ActiveGate.SetAPIURL(dk.APIURL())
	dk.Spec.ActiveGate.SetName(dk.Name)
	dk.Spec.ActiveGate.SetAutomaticTLSCertificate(dk.FF().IsActiveGateAutomaticTLSCertificate())
	dk.Spec.ActiveGate.SetExtensionsDependency(dk.Extensions().IsAnyEnabled())

	return &activegate.ActiveGate{
		Spec:   &dk.Spec.ActiveGate,
		Status: &dk.Status.ActiveGate,
	}
}
