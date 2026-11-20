// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warnDisabledTLSForPrometheus = "The DynaKube's `spec.tls.mode` is set to `Off`. This affects all existing PrometheusMonitoring resources, as well as any created in the future, that reference this DynaKube via `spec.dynaKubeName`: their metrics will be sent over unencrypted in-cluster connections between Prometheus components. Consider setting `spec.tls.mode` to `Automatic` to secure this traffic."
)

func disabledTLSForPrometheus(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.IsTLSOff() {
		return warnDisabledTLSForPrometheus
	}

	return ""
}
