package edgeconnect

import "time"

type OauthClientStatus int

type Metadata struct {
	OauthClientStatus string     `json:"oauthClientStatus"`
	Instances         []Instance `json:"instances"`
}

type ModificationInfo struct {
	LastModifiedTime *time.Time `json:"lastModifiedTime,omitempty"`
	LastModifiedBy   string     `json:"lastModifiedBy,omitempty"`
}

type Instance struct {
	Version    string `json:"version,omitempty"`
	InstanceId string `json:"instanceId,omitempty"`
}

type GetResponse struct {
	ModificationInfo           ModificationInfo `json:"modificationInfo"`
	Metadata                   Metadata         `json:"metadata"`
	ID                         string           `json:"id,omitempty"`
	Name                       string           `json:"name"`
	OauthClientId              string           `json:"oauthClientId"`
	HostPatterns               []string         `json:"hostPatterns"`
	ManagedByDynatraceOperator bool             `json:"managedByDynatraceOperator,omitempty"`
}

type ListResponse struct {
	EdgeConnects []GetResponse `json:"edgeConnects"`
	TotalCount   int           `json:"totalCount"`
}

type CreateResponse struct {
	ModificationInfo           ModificationInfo `json:"modificationInfo"`
	Metadata                   Metadata         `json:"metadata"`
	ID                         string           `json:"id,omitempty"`
	Name                       string           `json:"name"`
	OauthClientId              string           `json:"oauthClientId"`
	OauthClientSecret          string           `json:"oauthClientSecret"`
	OauthClientResource        string           `json:"oauthClientResource"`
	HostPatterns               []string         `json:"hostPatterns"`
	ManagedByDynatraceOperator bool             `json:"managedByDynatraceOperator,omitempty"`
}

type Request struct {
	Name                       string   `json:"name"`
	OauthClientId              string   `json:"oauthClientId,omitempty"`
	HostPatterns               []string `json:"hostPatterns"`
	ManagedByDynatraceOperator bool     `json:"managedByDynatraceOperator,omitempty"`
}

func NewRequest(name string, hostPatterns []string, oauthClientId string) *Request {
	return &Request{
		Name:                       name,
		HostPatterns:               hostPatterns,
		OauthClientId:              oauthClientId,
		ManagedByDynatraceOperator: true,
	}
}
