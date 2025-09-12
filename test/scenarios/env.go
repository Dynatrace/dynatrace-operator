//go:build e2e

package scenarios

import "os"

const HelmChartTagEnvVar = "USE_HELM_CHART_TAG"

func InstallViaHelm() bool {
	return os.Getenv(HelmChartTagEnvVar) != ""
}
