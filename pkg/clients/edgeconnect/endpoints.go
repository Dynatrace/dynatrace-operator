package edgeconnect

import "fmt"

// EdgeConnect API

func (c *client) getEdgeConnectApiUrl() string {
	return fmt.Sprintf("%s/platform/app-engine/edge-connect/v1", c.baseURL)
}

func (c *client) getEdgeConnectsUrl() string {
	return fmt.Sprintf("%s/edge-connects", c.getEdgeConnectApiUrl())
}

func (c *client) getEdgeConnectUrl(id string) string {
	return fmt.Sprintf("%s/%s", c.getEdgeConnectsUrl(), id)
}

// Environment API

func (c *client) getEnvironmentApiUrl() string {
	return fmt.Sprintf("%s/platform/classic/environment-api/v2", c.baseURL)
}

func (c *client) getSettingsObjectsUrl() string {
	return fmt.Sprintf("%s/settings/objects", c.getEnvironmentApiUrl())
}

func (c *client) getSettingsObjectsIdUrl(objectId string) string {
	return fmt.Sprintf("%s/%s", c.getSettingsObjectsUrl(), objectId)
}
