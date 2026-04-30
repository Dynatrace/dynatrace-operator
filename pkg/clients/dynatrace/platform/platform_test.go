package platform

import (
	"errors"
	"testing"

	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTenantPhase(t *testing.T) {
	createClient := func(t *testing.T, expectedPhase int, expectedErr error) *ClientImpl {
		t.Helper()

		req := coremock.NewRequest(t)
		req.EXPECT().Execute(new(tenantPhaseResponse)).Run(func(obj any) {
			if expectedErr == nil {
				obj.(*tenantPhaseResponse).PhaseID = expectedPhase
			}
		}).Return(expectedErr).Once()

		coreClient := coremock.NewClient(t)
		coreClient.EXPECT().GET(t.Context(), tenantPhasePath).Return(req).Once()

		return NewClient(coreClient)
	}

	t.Run("returns phase from response", func(t *testing.T) {
		client := createClient(t, 2, nil)

		phase, err := client.GetTenantPhase(t.Context())

		require.NoError(t, err)
		assert.Equal(t, 2, phase)
	})

	t.Run("returns zero and error on API failure", func(t *testing.T) {
		client := createClient(t, 0, errors.New("api error"))

		phase, err := client.GetTenantPhase(t.Context())

		require.Error(t, err)
		assert.Zero(t, phase)
	})
}
