package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
)

func (dk *DynaKube) LogMonitoring() *logmonitoring.LogMonitoring {
	dk.Spec.LogMonitoring.SetName(dk.Name)

	return &logmonitoring.LogMonitoring{
		Spec: &dk.Spec.LogMonitoring,
	}
}

func (dk *DynaKube) LogMonitoringTemplates() logmonitoring.TemplateSpec {
	return dk.Spec.Templates.LogMonitoring
}
