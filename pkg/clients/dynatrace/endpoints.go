package dynatrace

func (dtc *dynatraceClient) getProcessModuleConfigURL() string {
	return dtc.url + "/v1/deployment/installer/agent/processmoduleconfig?sections=general,agentType"
}

func (dtc *dynatraceClient) getTokensLookupURL() string {
	return dtc.url + "/v2/apiTokens/lookup"
}
