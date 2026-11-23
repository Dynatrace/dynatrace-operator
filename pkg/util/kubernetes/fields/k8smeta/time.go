// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8smeta

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TimeoutReached reports whether threshold has elapsed since timestamp.
// A nil or zero last is treated as reached.
func TimeoutReached(timestamp *metav1.Time, threshold time.Duration) bool {
	return timestamp.IsZero() || time.Since(timestamp.Time) >= threshold
}
