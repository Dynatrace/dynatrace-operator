// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package discoverycache

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestDisableCacheForTest(t *testing.T) {
	cache := New("test", func(context.Context, Client) (int, error) {
		return 0, nil
	})

	getConfig = func() (*rest.Config, error) {
		panic("unexpected call")
	}

	newClient = func(*rest.Config) (Client, error) {
		panic("unexpected call")
	}

	t.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	actualUndo := cache.DisableForTest(123)

	assert.Equal(t, 123, cache.value)
	assert.True(t, cache.disableLookup)
	assert.NotPanics(t, func() { cache.Fetch() })

	nopUndo := cache.DisableForTest(321)
	assert.Equal(t, 321, cache.value)
	assert.True(t, cache.disableLookup)

	nopUndo()

	assert.Equal(t, 321, cache.value)
	assert.True(t, cache.disableLookup)
	assert.NotPanics(t, func() { cache.Fetch() })

	actualUndo()

	assert.Equal(t, 0, cache.value)
	assert.False(t, cache.disableLookup)
	assert.Panics(t, func() { cache.Fetch() })
}

func TestFetch(t *testing.T) {
	var counter atomic.Int64

	getConfig = func() (*rest.Config, error) {
		counter.Add(1)

		return &rest.Config{}, nil
	}

	newClient = func(*rest.Config) (Client, error) {
		return fakeclientset.NewClientset().Discovery(), nil
	}

	t.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	cache := New("test", func(context.Context, Client) (int, error) {
		return 123, nil
	})

	synctest.Test(t, func(t *testing.T) {
		const iterations = 1_000
		results := make([]int, iterations)
		for i := range iterations {
			go func() {
				results[i] = cache.Fetch()
			}()
		}

		synctest.Wait()

		assert.Equal(t, slices.Repeat([]int{123}, iterations), results)
		assert.Equal(t, int64(1), counter.Load())

		time.Sleep(refreshInterval + 1)
		_ = cache.Fetch()
		assert.Equal(t, int64(2), counter.Load())
	})
}

func TestGetMinorVersionOnError(t *testing.T) {
	var counter atomic.Int64

	getConfig = func() (*rest.Config, error) {
		counter.Add(1)

		return &rest.Config{}, nil
	}

	newClient = func(*rest.Config) (Client, error) {
		return fakeclientset.NewClientset().Discovery(), nil
	}

	t.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	cache := New("test", func(context.Context, Client) (int, error) {
		return 0, errors.New("bad")
	})

	synctest.Test(t, func(t *testing.T) {
		const iterations = 10
		for range iterations {
			go func() {
				cache.Fetch()
			}()
		}

		synctest.Wait()

		assert.Equal(t, int64(1), counter.Load())
	})
}

func BenchmarkGetMinorVersionUncached(b *testing.B) {
	getConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}

	sentinel := fakeclientset.NewClientset().Discovery()

	newClient = func(*rest.Config) (Client, error) {
		return sentinel, nil
	}

	b.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	cache := New(b.Name(), func(context.Context, Client) (int, error) {
		return 123, nil
	})

	for b.Loop() {
		if v := cache.Fetch(); v != 123 {
			b.Fatalf("got unexpected minor version: %d", v)
		}
		// reset clock
		cache.lastCheck = time.Time{}
	}
}

func BenchmarkGetMinorVersionCached(b *testing.B) {
	getConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}

	sentinel := fakeclientset.NewClientset().Discovery()

	newClient = func(*rest.Config) (Client, error) {
		return sentinel, nil
	}

	b.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	cache := New(b.Name(), func(context.Context, Client) (int, error) {
		return 123, nil
	})

	// Initialize cache
	_ = cache.Fetch()

	for b.Loop() {
		if v := cache.Fetch(); v != 123 {
			b.Fatalf("got unexpected minor version: %d", v)
		}
	}
}
