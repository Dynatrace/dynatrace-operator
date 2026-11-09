// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8ssecuritycontext

import (
	"strconv"

	"github.com/pkg/errors"
)

const (
	RootUser  int64 = 0
	RootGroup int64 = 0
)

// GetValidatedRunAsID will parse the provided string as valid user/group ID between 0 and `math.MaxInt32`.
// Returns `*int64` for compatibility with `corev1.SecurityContext`. Returns nil without an error if the value is empty.
func GetValidatedRunAsID(value string) (*int64, error) {
	if value == "" {
		return nil, nil //nolint:nilnil
	}

	// uint31 == positive int32 range
	parsed, err := strconv.ParseUint(value, 10, 31)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse provided runAs value %q, must be a positive int32", value)
	}

	return new(int64(parsed)), nil
}

// IsNonRoot reports whether the provided user and group IDs result in a non-root process.
// The user takes precedence over the group; if neither is set, nil is returned as it can't be determined.
// Returns `*bool` for compatibility with `corev1.SecurityContext`
func IsNonRoot(user, group *int64) *bool {
	if user != nil { // user takes precedence over group
		return new(*user != RootUser)
	}

	if group != nil { // if group is root, but no user is set, we are still "running as root"
		return new(*group != RootGroup)
	}

	return nil
}
