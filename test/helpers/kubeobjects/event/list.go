//go:build e2e

package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func List(t *testing.T, ctx context.Context, resource *resources.Resources, namespace string, listOpt resources.ListOption) corev1.EventList {
	var events corev1.EventList

	require.NoError(t, resource.WithNamespace(namespace).List(ctx, &events, listOpt))

	return events
}
