package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"

func (dk *DynaKube) IsExtensionsEnabled() bool {
	return dk.Spec.Extensions.Enabled
}

func (dk *DynaKube) ExtensionsTLSRefName() string {
	return dk.Spec.Templates.ExtensionExecutionController.TlsRefName
}

func (dk *DynaKube) ExtensionsNeedsSelfSignedTLS() bool {
	return dk.ExtensionsTLSRefName() == ""
}

func (dk *DynaKube) ExtensionsTLSSecretName() string {
	if dk.ExtensionsNeedsSelfSignedTLS() {
		return dk.Name + consts.ExtensionsSelfSignedTLSSecretSuffix
	}

	return dk.ExtensionsTLSRefName()
}

func (dk *DynaKube) ExtensionsExecutionControllerStatefulsetName() string {
	return dk.Name + "-extensions-controller"
}

func (dk *DynaKube) ExtensionsCollectorStatefulsetName() string {
	return dk.Name + "-extensions-collector"
}

func (dk *DynaKube) ExtensionsTokenSecretName() string {
	return dk.Name + "-extensions-token"
}
