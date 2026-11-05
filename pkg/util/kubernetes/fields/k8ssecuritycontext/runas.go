// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8ssecuritycontext

import (
	"context"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

// GetValidatedRunAsID will parse the provided string as valid user/group ID between 0 and `math.MaxInt32`.
// Returns `*int64` for compatibility with `corev1.SecurityContext`
func GetValidatedRunAsID(ctx context.Context, value string) *int64 {
	if value == "" {
		return nil
	}

	// uint31 == positive int32 range
	parsed, err := strconv.ParseUint(value, 10, 31)
	if err != nil {
		logd.FromContext(ctx).Error(err, "failed to parse annotation value, must be a positive int32", "value", value)

		return nil
	}

	return new(int64(parsed))
}
