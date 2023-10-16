package edgeconnect

import "fmt"

func (c *client) getEdgeConnectsUrl() string {
	return fmt.Sprintf("%s/edge-connects", c.baseURL)
}

func (c *client) getEdgeConnectUrl(id string) string {
	return fmt.Sprintf("%s/edge-connects/%s", c.baseURL, id)
}
