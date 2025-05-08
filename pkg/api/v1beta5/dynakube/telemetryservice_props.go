package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/telemetryingest"
)

func (dk *DynaKube) TelemetryIngest() *telemetryingest.TelemetryIngest {
	ts := &telemetryingest.TelemetryIngest{
		Spec: dk.Spec.TelemetryIngest,
	}
	ts.SetName(dk.Name)

	return ts
}
