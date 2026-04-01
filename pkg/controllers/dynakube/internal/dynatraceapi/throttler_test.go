package dynatraceapi

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestNewThrottler(t *testing.T) {
	throttler := NewThrottler()

	assert.NotNil(t, throttler)
	assert.Empty(t, throttler)
}

func TestThrottlerInit(t *testing.T) {
	t.Run("nil DynaKube returns no error", func(t *testing.T) {
		throttler := NewThrottler()

		err := throttler.Init(t.Context(), nil, nil, token.Tokens{})

		require.NoError(t, err)
	})

	t.Run("first reconcile initializes empty throttler, calcs prevConfig and disables throttling", func(t *testing.T) {
		throttler := NewThrottler()
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test-dynakube"},
		}

		err := throttler.Init(t.Context(), nil, dk, token.Tokens{})

		require.NoError(t, err)
		require.NotNil(t, dk.Status.Tenant.APIThrottler)
		assert.False(t, dk.Status.Tenant.APIThrottler.Enabled)
		assert.NotEmpty(t, dk.Status.Tenant.APIThrottler.PrevConfig)
		assert.Empty(t, dk.Status.Tenant.APIThrottler.LastRequestTimestamp)
	})

	t.Run("restores state from in-memory map", func(t *testing.T) {
		throttler := NewThrottler()
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test-dynakube"},
		}

		// Pre-populate the in-memory map with a previous state
		prevTimestamp := metav1.NewTime(time.Now())
		throttler[dk.Name] = &dynakube.APIThrottler{
			LastRequestTimestamp: prevTimestamp,
			PrevConfig:           "some-previous-hash",
			Enabled:              false,
		}

		err := throttler.Init(t.Context(), nil, dk, token.Tokens{})

		require.NoError(t, err)
		require.NotNil(t, dk.Status.Tenant.APIThrottler)
		// Restored timestamp must match what was stored
		assert.Equal(t, prevTimestamp.Time, dk.Status.Tenant.APIThrottler.LastRequestTimestamp.Time)
	})

	t.Run("throttling enabled when config unchanged and threshold not reached", func(t *testing.T) {
		throttler := NewThrottler()
		tokens := token.Tokens{}
		initialDK := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test-dynakube"},
		}
		initialDK.Spec.DynatraceAPIRequestThreshold = ptr.To(uint16(1))
		rerunDK := initialDK.DeepCopy()

		// First reconcile: computes and stores config hash, sets timestamp
		err := throttler.Init(t.Context(), nil, initialDK, tokens)
		require.NoError(t, err)
		throttler.Store(initialDK)
		initialStatus := initialDK.Status.Tenant.APIThrottler.DeepCopy()

		// Second reconcile: returns it, with no change to prevConfig
		err = throttler.Init(t.Context(), nil, rerunDK, tokens)

		require.NoError(t, err)
		require.NotNil(t, rerunDK.Status.Tenant.APIThrottler)
		assert.True(t, rerunDK.Status.Tenant.APIThrottler.Enabled)

		// only the APIThrottler.Enabled field should be different
		assert.NotEqual(t, initialStatus.Enabled, rerunDK.Status.Tenant.APIThrottler.Enabled)
		assert.Equal(t, initialStatus.PrevConfig, rerunDK.Status.Tenant.APIThrottler.PrevConfig)
		assert.Equal(t, initialStatus.LastRequestTimestamp, rerunDK.Status.Tenant.APIThrottler.LastRequestTimestamp)
	})

	t.Run("throttling disabled when threshold reached", func(t *testing.T) {
		throttler := NewThrottler()
		tokens := token.Tokens{}
		initialDK := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test-dynakube"},
		}
		initialDK.Spec.DynatraceAPIRequestThreshold = ptr.To(uint16(1))
		rerunDK := initialDK.DeepCopy()

		// Establish the correct config hash via a first init+store
		err := throttler.Init(t.Context(), nil, initialDK, tokens)
		require.NoError(t, err)
		throttler.Store(initialDK)

		// Push the stored timestamp beyond the threshold
		throttler[initialDK.Name].LastRequestTimestamp = metav1.NewTime(time.Now().Add(-2 * time.Minute))

		err = throttler.Init(t.Context(), nil, rerunDK, tokens)

		require.NoError(t, err)
		require.NotNil(t, rerunDK.Status.Tenant.APIThrottler)
		assert.False(t, rerunDK.Status.Tenant.APIThrottler.Enabled)
	})

	t.Run("throttling disabled when spec changed", func(t *testing.T) {
		throttler := NewThrottler()
		tokens := token.Tokens{}
		initialDK := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test-dynakube"},
		}
		initialDK.Spec.DynatraceAPIRequestThreshold = ptr.To(uint16(1))
		rerunDK := initialDK.DeepCopy()

		// Establish state
		err := throttler.Init(t.Context(), nil, initialDK, tokens)
		require.NoError(t, err)
		throttler.Store(initialDK)
		initialStatus := initialDK.Status.Tenant.APIThrottler.DeepCopy()

		// Second reconcile with a different spec (APIURL changed)
		rerunDK.Spec.NetworkZone = "networkzone"

		err = throttler.Init(t.Context(), nil, rerunDK, tokens)

		require.NoError(t, err)
		require.NotNil(t, rerunDK.Status.Tenant.APIThrottler)
		assert.False(t, rerunDK.Status.Tenant.APIThrottler.Enabled)
		assert.NotEqual(t, initialStatus.PrevConfig, rerunDK.Status.Tenant.APIThrottler.PrevConfig)
	})

	t.Run("throttling disabled when tokens changed", func(t *testing.T) {
		throttler := NewThrottler()
		initialDK := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test-dynakube"},
		}
		initialDK.Spec.DynatraceAPIRequestThreshold = ptr.To(uint16(60))
		rerunDK := initialDK.DeepCopy()

		originalTokens := token.Tokens{"api": &token.Token{Value: "original-token"}}

		// Establish state with original tokens
		err := throttler.Init(t.Context(), nil, initialDK, originalTokens)
		require.NoError(t, err)
		throttler.Store(initialDK)
		initialStatus := initialDK.Status.Tenant.APIThrottler.DeepCopy()

		// Second reconcile with different tokens
		changedTokens := token.Tokens{"api": &token.Token{Value: "changed-token"}}

		err = throttler.Init(t.Context(), nil, rerunDK, changedTokens)

		require.NoError(t, err)
		require.NotNil(t, rerunDK.Status.Tenant.APIThrottler)
		assert.False(t, rerunDK.Status.Tenant.APIThrottler.Enabled)
		assert.NotEqual(t, initialStatus.PrevConfig, rerunDK.Status.Tenant.APIThrottler.PrevConfig)
	})
}

func TestThrottlerStore(t *testing.T) {
	t.Run("sets timestamp and persists state to map", func(t *testing.T) {
		throttler := NewThrottler()
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test-dynakube"},
		}
		dk.Status.Tenant.APIThrottler = &dynakube.APIThrottler{}

		throttler.Store(dk)

		stored, ok := throttler[dk.Name]
		require.True(t, ok)
		assert.NotEmpty(t, stored.LastRequestTimestamp)
	})

	t.Run("initializes nil APIThrottler before storing", func(t *testing.T) {
		throttler := NewThrottler()
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test-dynakube"},
		}
		dk.Status.Tenant.APIThrottler = nil

		throttler.Store(dk)

		require.NotNil(t, dk.Status.Tenant.APIThrottler)
		_, ok := throttler[dk.Name]
		assert.True(t, ok)
	})
}

func TestThrottlerDelete(t *testing.T) {
	t.Run("removes existing entry", func(t *testing.T) {
		throttler := NewThrottler()
		throttler["test-dynakube"] = &dynakube.APIThrottler{}

		throttler.Delete("test-dynakube")

		_, ok := throttler["test-dynakube"]
		assert.False(t, ok)
	})

	t.Run("does not panic for non-existent entry", func(t *testing.T) {
		throttler := NewThrottler()

		assert.NotPanics(t, func() {
			throttler.Delete("does-not-exist")
		})
	})
}
