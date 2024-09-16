package dynakube

func (dk *DynaKube) NeedsLogModule() bool {
	return dk.Spec.LogModule.Enabled
}

func (dk *DynaKube) LogModuleTemplates() LogModuleTemplateSpec {
	return dk.Spec.Templates.LogModule
}
