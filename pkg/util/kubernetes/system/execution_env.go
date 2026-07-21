// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package system

import "os"

func IsRunLocally() bool {
	return os.Getenv("RUN_LOCAL") == "true"
}
