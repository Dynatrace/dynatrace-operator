// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package installer

import "context"

type Installer interface {
	InstallAgent(ctx context.Context, targetDir string) (bool, error)
}
