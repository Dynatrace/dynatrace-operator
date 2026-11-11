// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const warningSelfMonitoringDisabled = "We recommend enabling self-monitoring in the DynaKube to detect and resolve issues, and to accelerate any support investigation that may be needed."

func selfMonitoringDisabled(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.IsSelfMonitoringEnabled() {
		return warningSelfMonitoringDisabled
	}

	return ""
}
