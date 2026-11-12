// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// Package hostevent implements a Dynatrace API client to fetch host information and sending events for nodes.
package hostevent

import (
	"cmp"
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	MarkedForTerminationEvent = "MARKED_FOR_TERMINATION"

	eventsPath   = "/v1/events"
	hostsPath    = "/v1/entity/infrastructure/hosts"
	entitiesPath = "/v2/entities"
	loggerName   = "dtclient-hostevent"
)

type Client interface {
	// GetEntityIDForIP returns the host entity ID for a given IP address.
	GetEntityIDForIP(ctx context.Context, ip string) (string, error)
	// SendEvent posts an event to the Dynatrace API.
	SendEvent(ctx context.Context, event Event) error
}

type ClientImpl struct {
	apiClient   core.Client
	networkZone string
}

func NewClient(apiClient core.Client, networkZone string) *ClientImpl {
	return &ClientImpl{
		apiClient:   apiClient,
		networkZone: networkZone,
	}
}

type EntityNotFoundError struct {
	IP string
}

func (e EntityNotFoundError) Error() string {
	return "HOST entity not found for IP: " + e.IP
}

type HostResponse struct {
	EntityID      string   `json:"entityId"`
	NetworkZoneID string   `json:"networkZoneId"`
	IPAddresses   []string `json:"ipAddresses"`
}

// hostEntityMap maps IPs to their respective HOST entityID according to the Dynatrace API
type hostEntityMap map[string]string

// update adds or overwrites the IP-to-Entity mapping if the IP already existed.
// The reason we do this "overwrite check" is somewhat unknown, it used to be part of a "caching" logic, however that cache was actually never really used.
// Kept it "as is" mainly to not introduce new behavior, it is unknown how the API we use handles repeated IP usage. But it can be just dead code.
func (entityMap hostEntityMap) update(ctx context.Context, info HostResponse) {
	log := logd.FromContext(ctx)

	for _, ip := range info.IPAddresses {
		if oldEntityID, ok := entityMap[ip]; ok {
			log.Info("hosts mapping: duplicate IP, replacing HOST entity to 'newer' one", "ip", ip, "new", info.EntityID, "old", oldEntityID)
		}

		entityMap[ip] = info.EntityID
	}
}

// entitiesResponse is the subset of the /v2/entities response needed to resolve a HOST by IP.
type entitiesResponse struct {
	Entities []hostEntity `json:"entities"`
}

type hostEntity struct {
	EntityID    string `json:"entityId"`
	LastSeenTms int64  `json:"lastSeenTms"`
	Properties  struct {
		NetworkZone string   `json:"networkZone"`
		IPAddress   []string `json:"ipAddress"`
	} `json:"properties"`
}

// GetEntityIDForIP returns the host entity ID for a given IP address.
//
// It first asks the Monitored entities API (v2) for exactly the HOST entities carrying the IP,
// so the result size no longer depends on how many hosts the whole tenant has.
// If the token lacks `entities.read` (HTTP 403), it falls back to the legacy v1 hosts listing.
func (c *ClientImpl) GetEntityIDForIP(ctx context.Context, ip string) (string, error) {
	ctx, log := logd.NewFromContext(ctx, loggerName)

	if ip == "" {
		return "", errors.New("must provide IP")
	}

	entityID, err := c.getEntityIDForIPBySelector(ctx, ip)
	if core.IsForbidden(err) {
		log.Info("token cannot read /v2/entities (entities.read scope missing?), falling back to the v1 hosts listing", "ip", ip)

		return c.getEntityIDForIPByHostList(ctx, ip)
	}

	return entityID, err
}

func (c *ClientImpl) getEntityIDForIPBySelector(ctx context.Context, ip string) (string, error) {
	var resp entitiesResponse

	err := c.apiClient.GET(ctx, entitiesPath).
		WithQueryParams(map[string]string{
			"entitySelector": `type(HOST),ipAddress("` + ip + `")`,
			"from":           "now-30m",
			"fields":         "+lastSeenTms,+properties.networkZone,+properties.ipAddress",
		}).Execute(&resp)
	if err != nil {
		return "", setEndpointNotAvailable(err, entitiesPath)
	}

	// IPs get reused (e.g. node pools scaling in and out), so more than one HOST can match.
	// Order oldest first, so the most recently seen HOST wins in hostEntityMap.update.
	slices.SortStableFunc(resp.Entities, func(a, b hostEntity) int {
		return cmp.Compare(a.LastSeenTms, b.LastSeenTms)
	})

	hosts := make([]HostResponse, 0, len(resp.Entities))
	for _, e := range resp.Entities {
		hosts = append(hosts, HostResponse{
			EntityID:      e.EntityID,
			NetworkZoneID: e.Properties.NetworkZone,
			IPAddresses:   e.Properties.IPAddress,
		})
	}

	entityID := buildHostEntityMap(ctx, hosts, c.networkZone)[ip]
	if entityID == "" {
		return "", EntityNotFoundError{IP: ip}
	}

	return entityID, nil
}

// getEntityIDForIPByHostList is the legacy lookup: it lists HOST entities and filters client side.
// It only sees the first page, so on tenants with more hosts than pageSize it can miss the host.
func (c *ClientImpl) getEntityIDForIPByHostList(ctx context.Context, ip string) (string, error) {
	var hosts []HostResponse

	err := c.apiClient.GET(ctx, hostsPath).
		WithQueryParams(map[string]string{
			"relativeTime":   "30mins",
			"includeDetails": "false",
			"pageSize":       "4000", // Pagination work-around
		}).Execute(&hosts)
	if err != nil {
		return "", setEndpointNotAvailable(err, hostsPath)
	}

	entities := buildHostEntityMap(ctx, hosts, c.networkZone)

	entityID := entities[ip]
	if entityID == "" {
		return "", EntityNotFoundError{IP: ip}
	}

	return entityID, nil
}

func buildHostEntityMap(ctx context.Context, hosts []HostResponse, networkZone string) hostEntityMap {
	entities := make(hostEntityMap)

	for _, host := range hosts {
		// Dynatrace stores network zone names in lowercase, while spec.networkZone may use any case.
		if (networkZone != "" && strings.EqualFold(host.NetworkZoneID, networkZone)) ||
			(networkZone == "" && (host.NetworkZoneID == "default" || host.NetworkZoneID == "")) {
			entities.update(ctx, host)
		}
	}

	return entities
}

// Event struct which defines what event payload should contain
type Event struct {
	EventType     string           `json:"eventType"`
	Description   string           `json:"description"`
	Source        string           `json:"source"`
	AttachRules   EventAttachRules `json:"attachRules"`
	StartInMillis uint64           `json:"start"`
	EndInMillis   uint64           `json:"end"`
}

type EventAttachRules struct {
	EntityIDs []string `json:"entityIds"`
}

func NewMarkedForTerminationEvent(entityID, source, description string, timestamp time.Time) Event {
	ts := uint64(timestamp.UnixNano() / int64(time.Millisecond)) //nolint:gosec // won't overflow

	return Event{
		EventType:   MarkedForTerminationEvent,
		Description: description,
		Source:      source,
		AttachRules: EventAttachRules{
			EntityIDs: []string{entityID},
		},
		StartInMillis: ts,
		EndInMillis:   ts,
	}
}

// SendEvent posts an event to the Dynatrace API.
func (c *ClientImpl) SendEvent(ctx context.Context, event Event) error {
	ctx, _ = logd.NewFromContext(ctx, loggerName)

	if event.EventType == "" {
		return errors.New("no key set for eventType in eventData payload")
	}

	err := c.apiClient.
		POST(ctx, eventsPath).
		WithJSONBody(event).
		Execute(nil)

	return setEndpointNotAvailable(err, eventsPath)
}

func setEndpointNotAvailable(err error, endpoint string) error {
	if core.IsNotFound(err) {
		return &core.HTTPError{
			Message:    endpoint + " is not available on the tenant",
			StatusCode: http.StatusNotFound,
		}
	}

	return err
}
