package helm

import (
	"io/ioutil"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/hack/mage/config"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Version sets the Helm Charts version and appVersion
func Version() error {
	mg.SerialDeps(config.Init)
	if config.CHART_VERSION != "" {
		chartPath := config.HELM_CHART_DEFAULT_DIR + "Chart.yaml"

		chartBytes, err := ioutil.ReadFile(chartPath)
		if err != nil {
			return err
		}
		chartString := string(chartBytes)
		chartLines := strings.Split(chartString, "\n")

		for i := range chartLines {
			if strings.HasPrefix(chartLines[i], "version: ") {
				chartLines[i] = "version: " + config.CHART_VERSION
			} else if strings.HasPrefix(chartLines[i], "appVersion: ") {
				chartLines[i] = "appVersion: " + config.CHART_VERSION
			}
		}
		chartString = strings.Join(chartLines, "\n")
		chartBytes = []byte(chartString)
		err = ioutil.WriteFile(chartPath+".output", chartBytes, 0o644)
		if err != nil {
			return err
		}

		return sh.Run("mv", chartPath+".output", chartPath)
	}
	return nil
}
