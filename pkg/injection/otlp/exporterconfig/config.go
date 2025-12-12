package exporterconfig

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

var (
	log = logd.Get().WithName("otlp-exporter-configuration")
)

// ActiveGateCertDataName is the key used to store ActiveGate certificate data in the secret containing the ActiveGate Certificate for the OTLP exporter.
const ActiveGateCertDataName = "activegate-tls.crt"
