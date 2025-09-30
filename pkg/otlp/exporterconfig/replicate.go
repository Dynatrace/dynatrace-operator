package exporterconfig

import "fmt"

const (
	sourceSecretTemplate = "%s-otlp-exporter-config"
)

func GetSourceConfigSecretName(dkName string) string {
	return fmt.Sprintf(sourceSecretTemplate, dkName)
}
