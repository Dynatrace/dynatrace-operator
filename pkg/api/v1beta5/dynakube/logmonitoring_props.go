package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/logmonitoring"
)

func (dk *DynaKube) LogMonitoring() *logmonitoring.LogMonitoring {
	lm := &logmonitoring.LogMonitoring{
		Spec:         dk.Spec.LogMonitoring,
		TemplateSpec: dk.Spec.Templates.LogMonitoring,
	}
	lm.SetName(dk.Name)
	lm.SetHostAgentDependency(dk.OneAgent().IsDaemonsetRequired())

	return lm
}
