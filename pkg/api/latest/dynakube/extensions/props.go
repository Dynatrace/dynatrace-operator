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

func (e *Extensions) SetEnabled(enabled bool) {
	e.enabled = enabled
}

func (e *Extensions) IsEnabled() bool {
	return e.enabled
}

func (e *Extensions) GetTLSRefName() string {
	return e.Controller.TLSRefName
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
	return "dynatrace" + consts.ExtensionsControllerSuffix + "-" + consts.ExtensionsCollectorTargetPortName
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
