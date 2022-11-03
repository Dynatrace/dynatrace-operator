package troubleshoot

func getOneAgentImageEndpoint(troubleshootCtx *troubleshootContext) string {
	imageEndpoint := troubleshootCtx.dynakube.ApiUrlHost() + "/linux/oneagent"

	customImage := troubleshootCtx.dynakube.CustomOneAgentImage()
	version := troubleshootCtx.dynakube.Version()

	if customImage != "" {
		imageEndpoint = customImage
	} else if version != "" {
		imageEndpoint = imageEndpoint + ":" + version
	}

	logInfof("OneAgent image endpoint '%s'", imageEndpoint)
	return imageEndpoint
}

func getOneAgentCodeModulesImageEndpoint(troubleshootCtx *troubleshootContext) string {
	imageEndpoint := troubleshootCtx.dynakube.CodeModulesImage()

	if imageEndpoint != "" {
		logInfof("OneAgent codeModules image endpoint '%s'", imageEndpoint)
	} else {
		logInfof("OneAgent codeModules image endpoint is not used.")
	}
	return imageEndpoint
}

func getActiveGateImageEndpoint(troubleshootCtx *troubleshootContext) string {
	imageEndpoint := troubleshootCtx.dynakube.ApiUrlHost() + "/linux/activegate"

	customActiveGateImage := troubleshootCtx.dynakube.CustomActiveGateImage()
	if customActiveGateImage != "" {
		imageEndpoint = customActiveGateImage
	}

	logInfof("ActiveGate image endpoint '%s'", imageEndpoint)
	return imageEndpoint
}
