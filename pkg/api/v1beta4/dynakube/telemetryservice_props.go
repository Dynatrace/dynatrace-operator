package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/telemetryservice"
)

func (dk *DynaKube) TelemetryService() *telemetryservice.TelemetryService {
	ts := &telemetryservice.TelemetryService{
		Spec: dk.Spec.TelemetryService,
	}
	ts.SetName(dk.Name)

	return ts
}
