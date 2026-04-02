package dynatraceapi

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = logd.Get().WithName("dynatraceapi-throttler")
)

// Throttler is a map for keeping track of the Dynatrace API throttling mechanism for all DynaKubes in cluster.
type Throttler map[string]*dynakube.APIThrottler

func NewThrottler() Throttler {
	return make(Throttler)
}

// Init prepares the throttling state for a DynaKube at the start of a reconcile loop.
//
// It restores the previously stored APIThrottler from the in-memory map (if one exists), ensuring
// that the timestamp and config hash from the last reconcile are available for comparison.
//
// After restoring (or initializing) the state, it evaluates whether throttling should be active for
// the current reconcile by delegating to setThrottled, which compares the config hash and the time
// elapsed since the last API request against the configured threshold.
func (dt Throttler) Init(ctx context.Context, apiReader client.Reader, dk *dynakube.DynaKube, tokens token.Tokens) error {
	if dk == nil {
		return nil
	}

	status, ok := dt[dk.Name]
	if ok {
		dk.Status.Tenant.APIThrottler = status

		log.Debug("dynakube entry read", "dkName", dk.Name, "APIThrottler", status)
	} else {
		dk.Status.Tenant.APIThrottler = &dynakube.APIThrottler{}

		log.Debug("first reconcile, setting empty APIThrottler", "dkName", dk.Name, "APIThrottler", status)
	}

	return setThrottled(ctx, apiReader, dk, tokens)
}

func (dt Throttler) Store(dk *dynakube.DynaKube) {
	if dk.Status.Tenant.APIThrottler == nil {
		dk.Status.Tenant.APIThrottler = &dynakube.APIThrottler{}
	}

	if !dk.Status.Tenant.APIThrottler.Enabled {
		dk.Status.Tenant.APIThrottler.LastRequestTimestamp = metav1.Now()
	}

	dt[dk.Name] = dk.Status.Tenant.APIThrottler
}

func (dt Throttler) Delete(dkName string) {
	delete(dt, dkName)
}

func setThrottled(ctx context.Context, apiReader client.Reader, dk *dynakube.DynaKube, tokens token.Tokens) error {
	configHash, err := calcPrevConfigHash(ctx, apiReader, dk, tokens)
	if err != nil {
		log.Info("failed to calculate prev config hash", "dkName", dk.Name)

		return err
	}

	noTimestampSet := dk.Status.Tenant.APIThrottler.LastRequestTimestamp.IsZero()
	thresholdReached := time.Since(dk.Status.Tenant.APIThrottler.LastRequestTimestamp.Time) >= dk.APIRequestThreshold()
	configChanged := configHash != dk.Status.Tenant.APIThrottler.PrevConfig

	if noTimestampSet || thresholdReached || configChanged {
		log.Info("throttling not enabled", "dkName", dk.Name, "noTimestampSet", noTimestampSet, "thresholdReached", thresholdReached, "configChanged", configChanged)

		dk.Status.Tenant.APIThrottler.Enabled = false
		dk.Status.Tenant.APIThrottler.PrevConfig = configHash
	} else {
		log.Info("throttling enabled", "dkName", dk.Name)

		dk.Status.Tenant.APIThrottler.Enabled = true
	}

	return nil
}

// calcPrevConfigHash produces a base64-encoded hash that captures the inputs most likely to affect
// the result of a Dynatrace API call: the API tokens, the DynaKube spec, and relevant feature flags
// (e.g. no-proxy). When this hash changes between reconcile loops, throttling is bypassed so that
// the new configuration is applied immediately.
//
// The selection of inputs is best-effort and does not need to be exhaustive.
//
// The context and apiReader parameters are currently unused but are kept in the signature for future
// use, e.g. to fetch referenced Secrets (proxy, trusted CAs) and include them in the hash.
func calcPrevConfigHash(_ context.Context, _ client.Reader, dk *dynakube.DynaKube, tokens token.Tokens) (string, error) {
	tokenHash, err := hasher.GenerateSecureHash(tokens)
	if err != nil {
		return "", err
	}

	// TODO: Fine tune, so only relevant parts of the spec changes we react on
	// TODO: Fine tune, so we also check referenced configs that affect the API (example: proxy, trustedCAs, etc)
	specHash, err := hasher.GenerateHash(dk.Spec)
	if err != nil {
		return "", err
	}

	relevantFFs := []string{
		dk.FF().GetNoProxy(),
	}

	ffHash, err := hasher.GenerateHash(relevantFFs)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString([]byte(tokenHash + specHash + ffHash)), nil
}
