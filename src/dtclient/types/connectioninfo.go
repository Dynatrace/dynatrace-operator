package types

type ConnectionInfo struct {
	CommunicationHosts              []CommunicationHost
	TenantUUID                      string
	TenantToken                     string
	FormattedCommunicationEndpoints string
}

// CommunicationHost => struct of connection endpoint
type CommunicationHost struct {
	Protocol string
	Host     string
	Port     uint32
}
