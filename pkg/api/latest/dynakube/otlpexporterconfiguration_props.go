// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"

func (dk *DynaKube) OTLPExporterConfiguration() *otlp.ExporterConfiguration {
	return otlp.NewExporterConfiguration(dk.Spec.OTLPExporterConfiguration, dk.GetResourceAttributes())
}
