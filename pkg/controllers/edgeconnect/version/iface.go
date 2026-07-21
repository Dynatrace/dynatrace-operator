// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import "context"

type versionStatusUpdater interface {
	Name() string
	RequiresReconcile() bool
	Update(ctx context.Context) error
}
