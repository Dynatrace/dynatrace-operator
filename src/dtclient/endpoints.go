package dtclient

import "fmt"

func (dtc *dynatraceClient) getAgentUrl(os, installerType, flavor, arch, version, technologies string) string {
	return fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/version/%s?flavor=%s&arch=%s&include=%s&bitness=64",
		dtc.url, os, installerType, version, flavor, technologies, arch)
}

func (dtc *dynatraceClient) getLatestAgentUrl(os, installerType, flavor, arch string) string {
	return fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest?bitness=64&flavor=%s&arch=%s",
		dtc.url, os, installerType, flavor, arch)
}

func (dtc *dynatraceClient) getAgentVersionsUrl(os, installerType, flavor, arch string) string {
	return fmt.Sprintf("%s/v1/deployment/installer/agent/versions/%s/%s?flavor=%s&arch=%s",
		dtc.url, os, installerType, flavor, arch)
}

func (dtc *dynatraceClient) getLatestAgentVersionUrl(os string, installerType string) string {
	return fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest/metainfo", dtc.url, os, installerType)
}

func (dtc *dynatraceClient) getConnectionInfoUrl() string {
	return fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dtc.url)
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
