package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/telemetryservice"
)

func (dk *DynaKube) TelemetryService() *telemetryservice.TelemetryService {
	return &telemetryservice.TelemetryService{
		Spec: dk.Spec.TelemetryService,
	}
}
