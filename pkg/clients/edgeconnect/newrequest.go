//go:build !e2e

package edgeconnect

func newRequest(name string, hostPatterns []string, hostMappings []HostMapping, oauthClientId string, _ ...bool) *Request {
	return &Request{
		Name:                       name,
		HostPatterns:               hostPatterns,
		HostMappings:               hostMappings,
		OauthClientId:              oauthClientId,
		ManagedByDynatraceOperator: true,
	}
}
