package edgeconnect

import "fmt"

func (c *client) getEdgeConnectApiUrl() string {
	return fmt.Sprintf("%s/%s", c.baseURL, "platform/app-engine/edge-connect/v1")
}

func (c *client) getEdgeConnectsUrl() string {
	return fmt.Sprintf("%s/%s", c.getEdgeConnectApiUrl(), "edge-connects")
}

func (c *client) getEdgeConnectUrl(id string) string {
	return fmt.Sprintf("%s/edge-connects/%s", c.baseURL, id)
}

func (c *client) getEnvironmentApiUrl() string {
	return fmt.Sprintf("%s/%s", c.baseURL, "platform/classic/environment-api/v2")
}

func (c *client) getSettingsObjectsUrl() string {
	return fmt.Sprintf("%s/settings/objects", c.getEnvironmentApiUrl())
}

func (c *client) getSettingsObjectsIdUrl(objectId string) string {
	return fmt.Sprintf("%s/%s", c.getSettingsObjectsUrl(), objectId)
}
