package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/telemetryservice"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (dk *DynaKube) TelemetryService() *telemetryservice.TelemetryService {
	ts := &telemetryservice.TelemetryService{
		Spec: dk.Spec.TelemetryService,
	}
	ts.SetName(dk.Name)

	return ts
}

func (dk *DynaKube) TelemetryApiCredentialsSecretName() *metav1.LabelSelector {
	return &dk.Spec.MetadataEnrichment.NamespaceSelector
}
