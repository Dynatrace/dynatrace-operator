package dynatrace

import "fmt"

func (dtc *dynatraceClient) getAgentURL(os, installerType, flavor, arch, version string, technologies []string, skipMetadata bool) string {
	url := fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/version/%s?flavor=%s&arch=%s&bitness=64&skipMetadata=%t",
		dtc.url, os, installerType, version, flavor, arch, skipMetadata)

	return appendTechnologies(url, technologies)
}

func (dtc *dynatraceClient) getLatestAgentURL(os, installerType, flavor, arch string, technologies []string, skipMetadata bool) string {
	url := fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest?bitness=64&flavor=%s&arch=%s&skipMetadata=%t",
		dtc.url, os, installerType, flavor, arch, skipMetadata)

	return appendTechnologies(url, technologies)
}

func (dtc *dynatraceClient) getAgentVersionsURL(os, installerType, flavor, arch string) string {
	if arch != "" {
		return fmt.Sprintf("%s/v1/deployment/installer/agent/versions/%s/%s?flavor=%s&arch=%s",
			dtc.url, os, installerType, flavor, arch)
	}

	return fmt.Sprintf("%s/v1/deployment/installer/agent/versions/%s/%s?flavor=%s",
		dtc.url, os, installerType, flavor)
}

func (dtc *dynatraceClient) getOneAgentConnectionInfoURL() string {
	if dtc.networkZone != "" {
		return fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo?networkZone=%s&defaultZoneFallback=true", dtc.url, dtc.networkZone)
	}

	return dtc.url + "/v1/deployment/installer/agent/connectioninfo"
}

func appendTechnologies(url string, technologies []string) string {
	for _, tech := range technologies {
		url = fmt.Sprintf("%s&include=%s", url, tech)
	}

	return url
}
