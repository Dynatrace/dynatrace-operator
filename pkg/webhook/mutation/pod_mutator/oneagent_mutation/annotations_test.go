package oneagent_mutation

import "reflect"

const (
	testFlavor       = "testFlavor"
	testTechnologies = "testTech"
	testInstallPath  = "testInstallPath"
	testInstallerURL = "testInstallerUrl"
)

func getTestInstallerInfo() installerInfo {
	return installerInfo{
		flavor:       testFlavor,
		technologies: testTechnologies,
		installPath:  testInstallPath,
		installerURL: testInstallerURL,
	}
}

func getInstallerInfoFieldCount() int {
	return reflect.TypeOf(installerInfo{}).NumField()
}
