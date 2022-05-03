package dtclient

import "fmt"

func (dtc *dynatraceClient) getAgentUrl(os, installerType, flavor, arch, version string, technologies []string) string {
	url := fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/version/%s?flavor=%s&arch=%s&bitness=64",
		dtc.url, os, installerType, version, flavor, arch)
	return appendTechnologies(url, technologies)
}

func (dtc *dynatraceClient) getLatestAgentUrl(os, installerType, flavor, arch string, technologies []string) string {
	url := fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest?bitness=64&flavor=%s&arch=%s",
		dtc.url, os, installerType, flavor, arch)
	return appendTechnologies(url, technologies)
}

func (dtc *dynatraceClient) getAgentVersionsUrl(os, installerType, flavor, arch string) string {
	return fmt.Sprintf("%s/v1/deployment/installer/agent/versions/%s/%s?flavor=%s&arch=%s",
		dtc.url, os, installerType, flavor, arch)
}

func (dtc *dynatraceClient) getOneAgentConnectionInfoUrl() string {
	return fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dtc.url)
}

func (dtc *dynatraceClient) getActiveGateConnectionInfoUrl() string {
	return fmt.Sprintf("%s/v1/deployment/installer/gateway/connectioninfo", dtc.url)
}

func (dtc *dynatraceClient) getHostsUrl() string {
	return fmt.Sprintf("%s/v1/entity/infrastructure/hosts?includeDetails=false", dtc.url)
}

func (dtc *dynatraceClient) getEntitiesUrl() string {
	return fmt.Sprintf("%s/v2/entities", dtc.url)
}

func (dtc *dynatraceClient) getSettingsUrl(validate bool) string {
	validationQuery := ""
	if !validate {
		validationQuery = "?validateOnly=false"
	}
	return fmt.Sprintf("%s/v2/settings/objects%s", dtc.url, validationQuery)
}

func (dtc *dynatraceClient) getProcessModuleConfigUrl() string {
	return fmt.Sprintf("%s/v1/deployment/installer/agent/processmoduleconfig", dtc.url)
}

func (dtc *dynatraceClient) getEventsUrl() string {
	return fmt.Sprintf("%s/v1/events", dtc.url)
}

func (dtc *dynatraceClient) getTokensLookupUrl() string {
	return fmt.Sprintf("%s/v1/tokens/lookup", dtc.url)
}

func appendTechnologies(url string, technologies []string) string {
	for _, tech := range technologies {
		url = fmt.Sprintf("%s&include=%s", url, tech)
	}
	return url
}
