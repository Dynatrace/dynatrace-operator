package dynakube

const (
	logModuleDaemonSetSuffix = "-logmodule"
)

func (dk *DynaKube) NeedsLogModule() bool {
	return dk.Spec.LogModule.Enabled
}

func (dk *DynaKube) LogModuleTemplates() LogModuleTemplateSpec {
	return dk.Spec.Templates.LogModule
}

func (dk *DynaKube) LogModuleDaemonSetName() string {
	return dk.Name + logModuleDaemonSetSuffix
}

func (dk *DynaKube) LogModuleNodeSelector() map[string]string {
	return dk.Spec.Templates.LogModule.NodeSelector
}
