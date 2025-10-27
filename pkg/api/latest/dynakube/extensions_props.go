package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
)

func (dk *DynaKube) Extensions() *extensions.Extensions {
	ext := &extensions.Extensions{
		ExecutionController: &dk.Spec.Templates.ExtensionExecutionController,
	}
	if dk.Spec.Extensions != nil {
		ext.Databases = dk.Spec.Extensions.Databases
	}

	// Set required fields for getters that may be called when extensions are disabled.
	ext.SetName(dk.Name)
	ext.SetNamespace(dk.Namespace)
	ext.SetPrometheusEnabled(dk.Spec.Extensions != nil && dk.Spec.Extensions.Prometheus != nil)

	return ext
}
