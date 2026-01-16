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

func (dtc *dynatraceClient) getLatestAgentVersionURL(os, installerType, flavor, arch string) string {
	if arch != "" {
		return fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest/metainfo?bitness=64&flavor=%s&arch=%s",
			dtc.url, os, installerType, flavor, arch)
	}

	return fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest/metainfo?bitness=64&flavor=%s",
		dtc.url, os, installerType, flavor)
}

func (dtc *dynatraceClient) getLatestActiveGateVersionURL(os string) string {
	return fmt.Sprintf("%s/v1/deployment/installer/gateway/%s/latest/metainfo",
		dtc.url, os)
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

func (dtc *dynatraceClient) getActiveGateConnectionInfoURL() string {
	return dtc.url + "/v1/deployment/installer/gateway/connectioninfo"
}

func (dtc *dynatraceClient) getHostsURL() string {
	return dtc.url + "/v1/entity/infrastructure/hosts?relativeTime=30mins&includeDetails=false"
}

func (dtc *dynatraceClient) getProcessModuleConfigURL() string {
	return dtc.url + "/v1/deployment/installer/agent/processmoduleconfig?sections=general,agentType"
}

func (dtc *dynatraceClient) getEventsURL() string {
	return dtc.url + "/v1/events"
}

func (dtc *dynatraceClient) getTokensLookupURL() string {
	return dtc.url + "/v2/apiTokens/lookup"
}

func (dtc *dynatraceClient) getActiveGateAuthTokenURL() string {
	return dtc.url + "/v2/activeGateTokens"
}

func (dtc *dynatraceClient) getLatestOneAgentImageURL() string {
	return dtc.url + "/v1/deployment/image/agent/oneAgent/latest"
}

func (dtc *dynatraceClient) getLatestCodeModulesImageURL() string {
	return dtc.url + "/v1/deployment/image/agent/codeModules/latest"
}

func (dtc *dynatraceClient) getLatestActiveGateImageURL() string {
	return dtc.url + "/v1/deployment/image/gateway/latest"
}

func appendTechnologies(url string, technologies []string) string {
	for _, tech := range technologies {
		url = fmt.Sprintf("%s&include=%s", url, tech)
	}

	return url
}
