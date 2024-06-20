package edgeconnect

// EdgeConnect API

func (c *client) getEdgeConnectApiUrl() string {
	return c.baseURL + "/platform/app-engine/edge-connect/v1"
}

func (c *client) getEdgeConnectsUrl() string {
	return c.getEdgeConnectApiUrl() + "/edge-connects"
}

func (c *client) getEdgeConnectUrl(id string) string {
	return c.getEdgeConnectsUrl() + "/" + id
}

// Environment API

func (c *client) getEnvironmentApiUrl() string {
	return c.baseURL + "/platform/classic/environment-api/v2"
}

func (c *client) getSettingsObjectsUrl() string {
	return c.getEnvironmentApiUrl() + "/settings/objects"
}

func (c *client) getSettingsObjectsIdUrl(objectId string) string {
	return c.getSettingsObjectsUrl() + "/" + objectId
}
