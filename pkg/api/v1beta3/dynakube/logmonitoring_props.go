package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
)

func (dk *DynaKube) LogMonitoring() *logmonitoring.LogMonitoring {
	lm := &logmonitoring.LogMonitoring{
		Spec: dk.Spec.LogMonitoring,
	}
	lm.SetName(dk.Name)

	return lm
}

func (dk *DynaKube) LogMonitoringTemplates() logmonitoring.TemplateSpec {
	return dk.Spec.Templates.LogMonitoring
}
