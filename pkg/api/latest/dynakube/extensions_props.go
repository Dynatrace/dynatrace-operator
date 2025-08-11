package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extension"
)

func (dk *DynaKube) Extensions() *extension.Extensions {
	ext := &extension.Extensions{
		ExecutionControllerSpec: &dk.Spec.Templates.ExtensionExecutionController,
	}
	// Set required fields for getters that may be called when extensions are disabled.
	ext.SetName(dk.Name)
	ext.SetNamespace(dk.Namespace)
	ext.SetEnabled(dk.Spec.Extensions != nil)

	return ext
}
