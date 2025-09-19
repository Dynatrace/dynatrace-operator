package dynatrace

import "fmt"

func (dtc *dynatraceClient) getAgentUrl(os, installerType, flavor, arch, version string, technologies []string, skipMetadata bool) string {
	url := fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/version/%s?flavor=%s&arch=%s&bitness=64&skipMetadata=%t",
		dtc.url, os, installerType, version, flavor, arch, skipMetadata)

	return appendTechnologies(url, technologies)
}

func (dtc *dynatraceClient) getLatestAgentUrl(os, installerType, flavor, arch string, technologies []string, skipMetadata bool) string {
	url := fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest?bitness=64&flavor=%s&arch=%s&skipMetadata=%t",
		dtc.url, os, installerType, flavor, arch, skipMetadata)

	return appendTechnologies(url, technologies)
}

func (dtc *dynatraceClient) getLatestAgentVersionUrl(os, installerType, flavor, arch string) string {
	if arch != "" {
		return fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest/metainfo?bitness=64&flavor=%s&arch=%s",
			dtc.url, os, installerType, flavor, arch)
	}

	return fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest/metainfo?bitness=64&flavor=%s",
		dtc.url, os, installerType, flavor)
}

func (dtc *dynatraceClient) getLatestActiveGateVersionUrl(os string) string {
	return fmt.Sprintf("%s/v1/deployment/installer/gateway/%s/latest/metainfo",
		dtc.url, os)
}

func (dtc *dynatraceClient) getAgentVersionsUrl(os, installerType, flavor, arch string) string {
	if arch != "" {
		return fmt.Sprintf("%s/v1/deployment/installer/agent/versions/%s/%s?flavor=%s&arch=%s",
			dtc.url, os, installerType, flavor, arch)
	}

	return fmt.Sprintf("%s/v1/deployment/installer/agent/versions/%s/%s?flavor=%s",
		dtc.url, os, installerType, flavor)
}

func (dtc *dynatraceClient) getOneAgentConnectionInfoUrl() string {
	if dtc.networkZone != "" {
		return fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo?networkZone=%s&defaultZoneFallback=true", dtc.url, dtc.networkZone)
	}

	return dtc.url + "/v1/deployment/installer/agent/connectioninfo"
}

func (dtc *dynatraceClient) getActiveGateConnectionInfoUrl() string {
	return dtc.url + "/v1/deployment/installer/gateway/connectioninfo"
}

func (dtc *dynatraceClient) getEntitiesUrl() string {
	return dtc.url + "/v2/entities"
}

func (dtc *dynatraceClient) getHostsUrl() string {
	return dtc.url + "/v1/entity/infrastructure/hosts?includeDetails=false"
}

func (dtc *dynatraceClient) getSettingsUrl(validate bool) string {
	validationQuery := ""
	if !validate {
		validationQuery = "?validateOnly=false"
	}

	return fmt.Sprintf("%s/v2/settings/objects%s", dtc.url, validationQuery)
}

func (dtc *dynatraceClient) getEffectiveSettingsUrl(validate bool) string {
	validationQuery := ""
	if !validate {
		validationQuery = "?validateOnly=false"
	}

	return fmt.Sprintf("%s/v2/settings/effectiveValues%s", dtc.url, validationQuery)
}

func (dtc *dynatraceClient) getProcessModuleConfigUrl() string {
	return dtc.url + "/v1/deployment/installer/agent/processmoduleconfig?sections=general,agentType"
}

func (dtc *dynatraceClient) getEventsUrl() string {
	return dtc.url + "/v1/events"
}

func (dtc *dynatraceClient) getTokensLookupUrl() string {
	return dtc.url + "/v2/apiTokens/lookup"
}

func (dtc *dynatraceClient) getActiveGateAuthTokenUrl() string {
	return dtc.url + "/v2/activeGateTokens"
}

func (dtc *dynatraceClient) getLatestOneAgentImageUrl() string {
	return dtc.url + "/v1/deployment/image/agent/oneAgent/latest"
}

func (dtc *dynatraceClient) getLatestCodeModulesImageUrl() string {
	return dtc.url + "/v1/deployment/image/agent/codeModules/latest"
}

func (dtc *dynatraceClient) getLatestActiveGateImageUrl() string {
	return dtc.url + "/v1/deployment/image/gateway/latest"
}

func appendTechnologies(url string, technologies []string) string {
	for _, tech := range technologies {
		url = fmt.Sprintf("%s&include=%s", url, tech)
	}

	return url
}
