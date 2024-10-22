package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
)

func (dk *DynaKube) LogMonitoring() *logmonitoring.LogMonitoring {
	lm := &logmonitoring.LogMonitoring{
		Spec:         dk.Spec.LogMonitoring,
		TemplateSpec: &dk.Spec.Templates.LogMonitoring,
	}
	lm.SetName(dk.Name)

	return lm
}
