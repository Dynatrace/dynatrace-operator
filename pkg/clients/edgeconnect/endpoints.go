package edgeconnect

// EdgeConnect API

func (c *client) getEdgeConnectAPIURL() string {
	return c.baseURL + "/platform/app-engine/edge-connect/v1"
}

func (c *client) getEdgeConnectsURL() string {
	return c.getEdgeConnectAPIURL() + "/edge-connects"
}

func (c *client) getEdgeConnectURL(id string) string {
	return c.getEdgeConnectsURL() + "/" + id
}

// Environment API

func (c *client) getEnvironmentAPIURL() string {
	return c.baseURL + "/platform/classic/environment-api/v2"
}

func (c *client) getSettingsObjectsURL() string {
	return c.getEnvironmentAPIURL() + "/settings/objects"
}

func (c *client) getSettingsObjectsIDURL(objectID string) string {
	return c.getSettingsObjectsURL() + "/" + objectID
}
