// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	errorPublicRegistryOverrideWithCustomImage = `The publicRegistryOverride field and a custom image (spec.imageRef.repository) are mutually exclusive. Remove one of them.`
)

func publicRegistryOverrideWithCustomImage(_ context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	if ec.Spec.PublicRegistryOverride != "" && ec.IsCustomImage() {
		return errorPublicRegistryOverrideWithCustomImage
	}

	return ""
}
