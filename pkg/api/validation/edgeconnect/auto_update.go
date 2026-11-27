// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	warningAutoUpdateDeprecated = "The EdgeConnect's `spec.autoUpdate` field is deprecated and will be removed in a future release."
)

func deprecatedAutoUpdate(_ context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	if ec.Spec.AutoUpdate != nil { //nolint:staticcheck // SA1019: intentionally reading deprecated field to warn about it
		return warningAutoUpdateDeprecated
	}

	return ""
}
