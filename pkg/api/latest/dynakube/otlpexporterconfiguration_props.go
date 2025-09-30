package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlpexporterconfiguration"
)

func (dk *DynaKube) OTLPExporterConfiguration() *otlpexporterconfiguration.OTLPExporterConfiguration {
	return &otlpexporterconfiguration.OTLPExporterConfiguration{
		Spec: dk.Spec.OTLPExporterConfiguration,
	}
}
