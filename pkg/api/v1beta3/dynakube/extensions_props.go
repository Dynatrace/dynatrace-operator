package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
)

func (dk *DynaKube) IsExtensionsEnabled() bool {
	return dk.Spec.Extensions != nil
}

func (dk *DynaKube) ExtensionsTLSRefName() string {
	return dk.Spec.Templates.ExtensionExecutionController.TlsRefName
}

func (dk *DynaKube) ExtensionsNeedsSelfSignedTLS() bool {
	return dk.ExtensionsTLSRefName() == ""
}

func (dk *DynaKube) ExtensionsTLSSecretName() string {
	if dk.ExtensionsNeedsSelfSignedTLS() {
		return dk.ExtensionsSelfSignedTLSSecretName()
	}

	return dk.ExtensionsTLSRefName()
}

func (dk *DynaKube) ExtensionsSelfSignedTLSSecretName() string {
	return dk.Name + consts.ExtensionsSelfSignedTLSSecretSuffix
}

func (dk *DynaKube) ExtensionsExecutionControllerStatefulsetName() string {
	return dk.Name + "-extensions-controller"
}

func (dk *DynaKube) ExtensionsTokenSecretName() string {
	return dk.Name + "-extensions-token"
}

func (dk *DynaKube) ExtensionsPortName() string {
	return "dynatrace" + consts.ExtensionsControllerSuffix + "-" + consts.ExtensionsDatasourceTargetPortName
}

func (dk *DynaKube) ExtensionsServiceNameFQDN() string {
	return dk.ExtensionsServiceName() + "." + dk.Namespace
}

func (dk *DynaKube) ExtensionsServiceName() string {
	return dk.Name + consts.ExtensionsControllerSuffix
}
