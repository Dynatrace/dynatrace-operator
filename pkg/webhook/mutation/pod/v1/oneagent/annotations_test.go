package oneagent

import "reflect"

const (
	testFlavor       = "testFlavor"
	testTechnologies = "testTech"
	testInstallPath  = "testInstallPath"
	testInstallerURL = "testInstallerUrl"
	testVersion      = "testVersion"
)

func getTestInstallerInfo() installerInfo {
	return installerInfo{
		flavor:       testFlavor,
		technologies: testTechnologies,
		installPath:  testInstallPath,
		installerURL: testInstallerURL,
		version:      testVersion,
	}
}

func getInstallerInfoFieldCount() int {
	return reflect.TypeOf(installerInfo{}).NumField()
}
