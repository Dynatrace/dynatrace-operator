package extensions

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
)

func (e *Extensions) SetName(name string) {
	e.name = name
}

func (e *Extensions) SetNamespace(namespace string) {
	e.namespace = namespace
}

func (e *Extensions) SetPrometheusEnabled(enabled bool) {
	e.prometheusEnabled = enabled
}

func (e *Extensions) IsPrometheusEnabled() bool {
	return e.prometheusEnabled
}

func (e *Extensions) IsDatabasesEnabled() bool {
	return len(e.Databases) > 0
}

func (e *Extensions) IsAnyEnabled() bool {
	return e.IsPrometheusEnabled() || e.IsDatabasesEnabled()
}

func (e *Extensions) GetTLSRefName() string {
	return e.ExecutionController.TLSRefName
}

func (e *Extensions) NeedsSelfSignedTLS() bool {
	return e.GetTLSRefName() == ""
}

func (e *Extensions) GetTLSSecretName() string {
	if e.NeedsSelfSignedTLS() {
		return e.GetSelfSignedTLSSecretName()
	}

	return e.GetTLSRefName()
}

func (e *Extensions) GetSelfSignedTLSSecretName() string {
	return e.name + consts.ExtensionsSelfSignedTLSSecretSuffix
}

func (e *Extensions) GetExecutionControllerStatefulsetName() string {
	return e.name + "-extensions-controller"
}

func (e *Extensions) GetTokenSecretName() string {
	return e.name + "-extensions-token"
}

func (e *Extensions) GetPortName() string {
	return "dynatrace" + consts.ExtensionsControllerSuffix + "-" + consts.ExtensionsDatasourceTargetPortName
}

func (e *Extensions) GetServiceNameFQDN() string {
	return e.GetServiceName() + "." + e.namespace
}

func (e *Extensions) GetServiceName() string {
	return e.name + consts.ExtensionsControllerSuffix
}

func (e *Extensions) GetDatabaseExecutorName(id string) string {
	return e.name + "-database-datasource-" + id
}
