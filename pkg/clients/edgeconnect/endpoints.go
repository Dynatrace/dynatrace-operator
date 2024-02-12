package edgeconnect

import "fmt"

func (c *client) getEdgeConnectsUrl() string {
	return c.baseURL + "/edge-connects"
}

func (c *client) getEdgeConnectUrl(id string) string {
	return fmt.Sprintf("%s/edge-connects/%s", c.baseURL, id)
}
