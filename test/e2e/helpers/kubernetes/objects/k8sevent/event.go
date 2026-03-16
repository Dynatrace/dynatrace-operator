//go:build e2e

package k8sevent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	eventsv1 "k8s.io/api/events/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func List(t *testing.T, ctx context.Context, resource *resources.Resources, namespace string, listOpt resources.ListOption) []eventsv1.Event {
	t.Helper()

	var events eventsv1.EventList

	require.NoError(t, resource.WithNamespace(namespace).List(ctx, &events, listOpt))

	return events.Items
}
