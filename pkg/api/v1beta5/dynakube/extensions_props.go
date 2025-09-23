package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/extensions"
)

func (dk *DynaKube) Extensions() *extensions.Extensions {
	ext := &extensions.Extensions{
		ExecutionControllerSpec: &dk.Spec.Templates.ExtensionExecutionController,
	}
	// Set required fields for getters that may be called when extensions are disabled.
	ext.SetName(dk.Name)
	ext.SetNamespace(dk.Namespace)
	ext.SetEnabled(dk.Spec.Extensions != nil)

	return ext
}
