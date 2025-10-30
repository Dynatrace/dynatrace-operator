package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"

func (dk *DynaKube) OTLPExporterConfiguration() *otlp.ExporterConfiguration {
	otlpExporterConfiguration := &otlp.ExporterConfiguration{
		Spec: dk.Spec.OTLPExporterConfiguration,
	}

	return otlpExporterConfiguration
}
