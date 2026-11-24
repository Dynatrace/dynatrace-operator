// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	errorFallbackModeWithDefaultNetworkZone = "The DynaKube's specification sets `networkZoneFallbackMode` while `networkZone` is \"default\". Remove `networkZoneFallbackMode` or use a different network zone."
)

func fallbackModeWithDefaultNetworkZone(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.NetworkZone == dynakube.NetworkZoneDefault && dk.Spec.NetworkZoneFallbackMode != "" {
		return errorFallbackModeWithDefaultNetworkZone
	}

	return ""
}
