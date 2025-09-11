//go:build e2e

package scenarios

import "os"

const installViaHelmEnvVar = "INSTALL_VIA_HELM"
const HelmChartTagEnvVar = "USE_HELM_CHART_TAG"

func InstallViaHelm() bool {
	if os.Getenv(installViaHelmEnvVar) == "true" && os.Getenv(HelmChartTagEnvVar) != "" {
		return true
	}

	return false
}
