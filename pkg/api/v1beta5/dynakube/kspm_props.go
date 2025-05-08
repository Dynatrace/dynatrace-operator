package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/kspm"

func (dk *DynaKube) KSPM() *kspm.Kspm {
	_kspm := &kspm.Kspm{
		Spec:                           dk.Spec.Kspm,
		Status:                         &dk.Status.Kspm,
		NodeConfigurationCollectorSpec: &dk.Spec.Templates.KspmNodeConfigurationCollector,
	}
	_kspm.SetName(dk.GetName())

	return _kspm
}
