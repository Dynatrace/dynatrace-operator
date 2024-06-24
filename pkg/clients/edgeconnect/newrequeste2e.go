//go:build e2e

package edgeconnect

func newRequest(name string, hostPatterns []string, hostMappings []HostMapping, oauthClientId string, managedByOperator ...bool) *Request {
	var managed bool
	if len(managedByOperator) == 1 {
		managed = managedByOperator[0]
	} else {
		panic("managedByOperator argument must be exactly single bool value")
	}

	return &Request{
		Name:                       name,
		HostPatterns:               hostPatterns,
		HostMappings:               hostMappings,
		OauthClientId:              oauthClientId,
		ManagedByDynatraceOperator: managed,
	}
}
