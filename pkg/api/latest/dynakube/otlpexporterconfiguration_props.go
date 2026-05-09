package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"

func (dk *DynaKube) OTLPExporterConfiguration() *otlp.ExporterConfiguration {
	return otlp.NewExporterConfiguration(dk.Spec.OTLPExporterConfiguration, dk.GetResourceAttributes())
}
