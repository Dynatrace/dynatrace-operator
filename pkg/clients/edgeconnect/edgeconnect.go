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
	Metadata         Metadata         `json:"metadata"`
}

type ListResponse struct {
	EdgeConnects []GetResponse `json:"edgeConnects"`
	TotalCount   int           `json:"totalCount"`
}

type CreateResponse struct {
	ID                  string           `json:"id,omitempty"`
	Name                string           `json:"name"`
	HostPatterns        []string         `json:"hostPatterns"`
	OauthClientId       string           `json:"oauthClientId"`
	OauthClientSecret   string           `json:"oauthClientSecret"`
	OauthClientResource string           `json:"oauthClientResource"`
	ModificationInfo    ModificationInfo `json:"modificationInfo"`
	Metadata            Metadata         `json:"metadata"`
}

type Request struct {
	Name          string   `json:"name"`
	HostPatterns  []string `json:"hostPatterns"`
	OauthClientId string   `json:"oauthClientId,omitempty"`
}
