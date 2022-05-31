package oneagent_mutation

const (
	testFlavor        = "testFlavor"
	testTechnologies  = "testTech"
	testInstallPath   = "testInstallPath"
	testInstallerURL  = "testInstallerUrl"
	testFailurePolicy = "testFailurePolicy"
)

func getTestInstallerInfo() installerInfo {
	return installerInfo{
		flavor:        testFlavor,
		technologies:  testTechnologies,
		installPath:   testInstallPath,
		installerURL:  testInstallerURL,
		failurePolicy: testFailurePolicy,
	}
}
