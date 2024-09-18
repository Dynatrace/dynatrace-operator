package dynakube

const (
	ExtensionsExecutionControllerStatefulsetName = "dynatrace-extensions-controller"
	ExtensionsCollectorStatefulsetName           = "dynatrace-extensions-collector"
)

func (dk *DynaKube) IsExtensionsEnabled() bool {
	return dk.Spec.Extensions.Enabled
}
