package edgeconnect

import edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"

// Client is the interface for the Dynatrace EdgeConnect REST API client.
type Client interface {
	// GetEdgeConnect return details of single edge connect
	GetEdgeConnect(edgeConnectId string) (GetResponse, error)

	// CreateEdgeConnect creates edge connect
	CreateEdgeConnect(name string, hostPatterns []string, hostMappings []edgeconnectv1alpha1.HostMapping, oauthClientId string) (CreateResponse, error)

	// UpdateEdgeConnect updates edge connect
	UpdateEdgeConnect(edgeConnectId, name string, hostPatterns []string, hostMappings []edgeconnectv1alpha1.HostMapping, oauthClientId string) error

	// DeleteEdgeConnect deletes edge connect
	DeleteEdgeConnect(edgeConnectId string) error

	// GetEdgeConnects returns list of edge connects
	GetEdgeConnects(name string) (ListResponse, error)

	// GetConnectionSetting returns a connection setting object by value uid
	GetConnectionSetting(uid string) (EnvironmentSetting, error)

	// CreateConnectionSetting creates a connection setting object
	CreateConnectionSetting(es EnvironmentSetting) error

	// UpdateConnectionSetting updates a connection setting object
	UpdateConnectionSetting(es EnvironmentSetting) error

	// DeleteConnectionSetting deletes a connection setting object
	DeleteConnectionSetting(objectId string) error
}
