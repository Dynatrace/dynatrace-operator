// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"
)

const (
	errorInvalidNetworkZone = "The DynaKube's specification has an invalid Network Zone value set. Make sure to remove forbidden characters (newline, tab, carriage return, null) from the Network Zone value in your custom resource."
)

const (
	errorFallbackModeWithDefaultNetworkZone = "The DynaKube's specification sets `networkZoneFallbackMode` while `networkZone` is \"default\". Remove `networkZoneFallbackMode` or use a different network zone."
)

func invalidNetworkZone(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if strings.ContainsAny(dk.Spec.NetworkZone, sanitize.InvalidCommandLineCharset) {
		return errorInvalidNetworkZone
	}

	return ""
}

func fallbackModeWithDefaultNetworkZone(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.NetworkZone == dynakube.NetworkZoneDefault && dk.Spec.NetworkZoneFallbackMode != "" {
		return errorFallbackModeWithDefaultNetworkZone
	}

	return ""
}
