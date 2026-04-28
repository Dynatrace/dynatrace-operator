package consts

const (
	AgCertificate                   = "custom-cas/agcrt.pem"
	AgCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	AgCertificateAndPrivateKeyField = "server.p12"
	AgSecretName                    = "ag-ca"
	TelemetryIngestTLSSecretName    = "telemetry-ingest-tls"
	DevRegistryPullSecretName       = "devregistry"

	DefaultCodeModulesImage = "ghcr.io/dynatrace/dynatrace-bootstrapper:snapshot"
	CodeModulesImageEnvVar  = "E2E_CODEMODULES_IMAGE"
)
