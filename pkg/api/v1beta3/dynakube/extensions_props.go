package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"

const (
	ExtensionsExecutionControllerStatefulsetName = "dynatrace-extensions-controller"
	ExtensionsCollectorStatefulsetName           = "dynatrace-extensions-collector"
)

func (dk *DynaKube) IsExtensionsEnabled() bool {
	return dk.Spec.Extensions.Prometheus.Enabled
}

func (dk *DynaKube) PrometheusEnabled() bool {
	return dk.Spec.Extensions.Prometheus.Enabled
}

func (dk *DynaKube) GetExtensionsTlsRefName() string {
	return dk.Spec.Templates.ExtensionExecutionController.TlsRefName
}

func (dk *DynaKube) GetExtensionsTlsSecretName() string {
	if dk.GetExtensionsTlsRefName() != "" {
		return dk.GetExtensionsTlsRefName()
	}

	return dk.Name + consts.ExtensionsSelfSignedTlsSecretSuffix
}
