package edgeconnect

// Client is the interface for the Dynatrace EdgeConnect REST API client.
type Client interface {
	// GetEdgeConnect return details of single edge connect
	GetEdgeConnect(edgeConnectId string) (GetResponse, error)

	// CreateEdgeConnect creates edge connect
	CreateEdgeConnect(name string, hostPatterns []string, hostMappings []HostMapping, oauthClientId string) (CreateResponse, error)

	// UpdateEdgeConnect updates edge connect
	UpdateEdgeConnect(edgeConnectId, name string, hostPatterns []string, hostMappings []HostMapping, oauthClientId string) error

	// DeleteEdgeConnect deletes edge connect
	DeleteEdgeConnect(edgeConnectId string) error

	// GetEdgeConnects returns list of edge connects
	GetEdgeConnects(name string) (ListResponse, error)

	// GetConnectionSetting returns a connection setting object by value name, namespace and kube-system namespace UID
	GetConnectionSetting(name, namespace, uid string) (EnvironmentSetting, error)

	// CreateConnectionSetting creates a connection setting object
	CreateConnectionSetting(es EnvironmentSetting) error

	// UpdateConnectionSetting updates a connection setting object
	UpdateConnectionSetting(es EnvironmentSetting) error

	// DeleteConnectionSetting deletes a connection setting object
	DeleteConnectionSetting(objectId string) error
}
