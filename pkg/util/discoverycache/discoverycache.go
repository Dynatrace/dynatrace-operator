// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package discoverycache

import (
	"context"
	"sync"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const refreshInterval = 5 * time.Minute

type Client interface {
	discovery.ServerResourcesInterface
	discovery.ServerVersionInterface
}

type Instance[T any] struct {
	mutex     sync.Mutex
	lastCheck time.Time
	// Not protected by mutex. Must only be set by DisableForTest.
	disableLookup bool

	name   string
	lookup func(context.Context, Client) (T, error)

	value T
}

func New[T any](name string, lookup func(context.Context, Client) (T, error)) *Instance[T] {
	return &Instance[T]{
		name:   name,
		lookup: lookup,
	}
}

func (i *Instance[T]) DisableForTest(value T) func() {
	if i.disableLookup {
		i.value = value
		// Set the value, but only undo the first time
		return func() {}
	}

	prevValue := i.value
	i.value = value
	i.disableLookup = true

	return func() {
		i.value = prevValue
		i.disableLookup = false
	}
}

// Fetch looks up the value using a discovery client.
// The value is cached and refreshed every 5 minutes, if requested.
//
// This function is safe to be called concurrently.
func (i *Instance[T]) Fetch() T {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	ctx, log := logd.NewFromContext(context.Background(), i.name)
	if err := i.refresh(ctx); err != nil {
		log.Error(err, "lookup failed")
	}

	return i.value
}

var (
	getConfig = config.GetConfig
	newClient = newDiscoveryClientForConfig
)

func newDiscoveryClientForConfig(c *rest.Config) (Client, error) {
	return discovery.NewDiscoveryClientForConfig(c)
}

func (i *Instance[T]) refresh(ctx context.Context) error {
	if i.disableLookup || !i.shouldRefresh() {
		return nil
	}

	// Set the timestamp early to ensure we only run this once ever time slot.
	// Any errors that occur will likely not go away in the timespan of a mutex lock/unlock,
	// so don't spam the logs with them.
	// IMPORTANT: All errors should include a stacktrace to improve visibility.
	i.lastCheck = time.Now()

	cfg, err := getConfig()
	if err != nil {
		return errors.Wrap(err, "load kubeconfig")
	}

	client, err := newClient(cfg)
	if err != nil {
		return errors.Wrap(err, "build discovery client")
	}

	value, err := i.lookup(ctx, client)
	if err != nil {
		return errors.WithStack(err)
	}

	i.value = value

	return nil
}

func (i *Instance[T]) shouldRefresh() bool {
	return i.lastCheck.IsZero() || time.Since(i.lastCheck) >= refreshInterval
}
