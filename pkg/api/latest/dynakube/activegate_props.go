package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
)

func (dk *DynaKube) ActiveGate() *activegate.ActiveGate {
	// The stored API URL is only used to derive the tenant image registry host, which is only
	// available on 2nd gen URLs. Pass the raw value so a 3rd gen URL yields a non-functional
	// registry host on purpose (see DynaKube.APIURLHost).
	dk.Spec.ActiveGate.SetAPIURL(dk.Spec.APIURL)
	dk.Spec.ActiveGate.SetName(dk.Name)
	dk.Spec.ActiveGate.SetAutomaticTLSCertificate(dk.FF().IsActiveGateAutomaticTLSCertificate())
	dk.Spec.ActiveGate.SetExtensionsDependency(dk.Extensions().IsAnyEnabled())

	return &activegate.ActiveGate{
		Spec:   &dk.Spec.ActiveGate,
		Status: &dk.Status.ActiveGate,
	}
}
