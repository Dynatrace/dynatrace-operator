package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlpexporterconfiguration"
)

func (dk *DynaKube) OTLPExporterConfiguration() *otlpexporterconfiguration.OTLPExporterConfiguration {
	otlpExporterConfiguration := &otlpexporterconfiguration.OTLPExporterConfiguration{
		Spec: dk.Spec.OTLPExporterConfiguration,
	}

	return otlpExporterConfiguration
}
