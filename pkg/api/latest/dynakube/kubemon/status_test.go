// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package kubemon_test

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/stretchr/testify/assert"
)

func TestStatus_IsZero(t *testing.T) {
	testCases := map[string]struct {
		status       kubemon.Status
		expectedZero bool
	}{
		"empty status is zero": {
			status:       kubemon.Status{},
			expectedZero: true,
		},
		"version status set, connection info empty is not zero": {
			status:       kubemon.Status{VersionStatus: status.VersionStatus{Version: "1.2.3"}},
			expectedZero: false,
		},
		"connection info set, version status empty is not zero": {
			status:       kubemon.Status{ConnectionInfo: communication.ConnectionInfo{TenantUUID: "abc123"}},
			expectedZero: false,
		},
		"both set is not zero": {
			status: kubemon.Status{
				VersionStatus:  status.VersionStatus{Version: "1.2.3"},
				ConnectionInfo: communication.ConnectionInfo{TenantUUID: "abc123"},
			},
			expectedZero: false,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedZero, testCase.status.IsZero())
		})
	}
}
