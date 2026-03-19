package dynatrace

func (dtc *dynatraceClient) getProcessModuleConfigURL() string {
	return dtc.url + "/v1/deployment/installer/agent/processmoduleconfig?sections=general,agentType"
}
