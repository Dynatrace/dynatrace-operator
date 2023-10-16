package edgeconnect

import "time"

type OauthClientStatus int

type Metadata struct {
	Instances         []Instance `json:"instances"`
	OauthClientStatus string     `json:"oauthClientStatus"`
}

type ModificationInfo struct {
	LastModifiedBy   string     `json:"lastModifiedBy,omitempty"`
	LastModifiedTime *time.Time `json:"lastModifiedTime,omitempty"`
}

type Instance struct {
	Version    string `json:"version,omitempty"`
	InstanceId string `json:"instanceId,omitempty"`
}

type GetResponse struct {
	ID               string           `json:"id,omitempty"`
	Name             string           `json:"name"`
	HostPatterns     []string         `json:"hostPatterns"`
	OauthClientId    string           `json:"oauthClientId"`
	ModificationInfo ModificationInfo `json:"modificationInfo"`
}

type CreateResponse struct {
	ID               string           `json:"id,omitempty"`
	Name             string           `json:"name"`
	HostPatterns     []string         `json:"hostPatterns"`
	OauthClientId    string           `json:"oauthClientId"`
	ModificationInfo ModificationInfo `json:"modificationInfo"`
}

type Request struct {
	Name          string   `json:"name"`
	HostPatterns  []string `json:"hostPatterns"`
	OauthClientId string   `json:"oauthClientId,omitempty"`
}
