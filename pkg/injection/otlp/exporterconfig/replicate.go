package exporterconfig

import "fmt"

const (
	sourceSecretTemplate      = "%s-otlp-exporter-config"
	sourceSecretCertsTemplate = "%s-otlp-exporter-certs"
)

func GetSourceConfigSecretName(dkName string) string {
	return fmt.Sprintf(sourceSecretTemplate, dkName)
}

func GetSourceCertsSecretName(dkName string) string {
	return fmt.Sprintf(sourceSecretCertsTemplate, dkName)
}
