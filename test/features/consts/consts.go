package consts

const (
	AgCertificate                   = "custom-cas/agcrt.pem"
	AgCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	AgCertificateAndPrivateKeyField = "server.p12"
	AgSecretName                    = "ag-ca"
	DevRegistryPullSecretName       = "devregistry"
	EecImageRepo                    = "478983378254.dkr.ecr.us-east-1.amazonaws.com/dynatrace/dynatrace-eec"
	EecImageTag                     = "1.303.0.20240930-183404"
	// LogMonitoringImageRepo          = "repository: public.ecr.aws/dynatrace/dynatrace-logmodule"
	LogMonitoringImageRepo = "us-central1-docker.pkg.dev/cloud-platform-207208/chmu/logmodule"
	LogMonitoringImageTag  = "latest"
)
