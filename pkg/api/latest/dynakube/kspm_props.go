package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"

func (dk *DynaKube) KSPM() *kspm.KSPM {
	_kspm := &kspm.KSPM{
		Spec:                           dk.Spec.KSPM,
		Status:                         &dk.Status.KSPM,
		NodeConfigurationCollectorSpec: &dk.Spec.Templates.KSPMNodeConfigurationCollector,
	}
	_kspm.SetName(dk.GetName())

	return _kspm
}
