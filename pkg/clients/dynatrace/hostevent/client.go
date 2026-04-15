// Package hostevent implements a Dynatrace API client to fetch host information and sending events for nodes.
package hostevent

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	MarkedForTerminationEvent = "MARKED_FOR_TERMINATION"

	eventsPath = "/v1/events"
	hostsPath  = "/v1/entity/infrastructure/hosts"
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

// Update adds or overwrites the IP-to-Entity mapping if the IP already existed.
// The reason we do this "overwrite check" is somewhat unknown, it used to be part of a "caching" logic, however that cache was actually never really used.
// Kept it "as is" mainly to not introduce new behavior, it is unknown how the API we use handles repeated IP usage. But it can be just dead code.
func (entityMap hostEntityMap) Update(ctx context.Context, info HostResponse) {
	log := logd.FromContext(ctx)

	for _, ip := range info.IPAddresses {
		if oldEntityID, ok := entityMap[ip]; ok {
			log.Info("hosts mapping: duplicate IP, replacing HOST entity to 'newer' one", "ip", ip, "new", info.EntityID, "old", oldEntityID)
		}

		entityMap[ip] = info.EntityID
	}
}

// GetEntityIDForIP returns the host entity ID for a given IP address.
func (c *ClientImpl) GetEntityIDForIP(ctx context.Context, ip string) (string, error) {
	if ip == "" {
		return "", errors.New("must provide IP")
	}

	var hosts []HostResponse

	err := c.apiClient.GET(ctx, hostsPath).
		WithQueryParams(map[string]string{
			"relativeTime":   "30mins",
			"includeDetails": "false",
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
		if (networkZone != "" && host.NetworkZoneID == networkZone) ||
			(networkZone == "" && (host.NetworkZoneID == "default" || host.NetworkZoneID == "")) {
			entities.Update(ctx, host)
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
