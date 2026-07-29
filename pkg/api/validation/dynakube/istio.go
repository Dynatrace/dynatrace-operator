// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"k8s.io/utils/ptr"
)

const (
	errorNoIstioInstalled = `No resources for istio available`
)

func isIstioNotInstalled(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if ptr.Deref(dk.Spec.EnableIstio, false) && !istio.IsInstalled(ctx, dv.apiReader) {
		return errorNoIstioInstalled
	}

	return ""
}
