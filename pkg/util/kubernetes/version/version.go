// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/discoverycache"
	"github.com/pkg/errors"
	k8sversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
)

var minorVersionCache = discoverycache.New("k8sversion", lookupMinorVersion)

// DisableCacheForTest disables the global version cache for testing and configures a static minor version that GetMinorVersion will return.
// The returned function can be used to undo this change. Calling this function multiple times with different inputs will update the minorVersion,
// but only the first undo function will revert changes.
//
// Example:
//
//	func TestFoo(t *testing.T) {
//		t.Cleanup(version.DisableCacheForTest(34))
//		...
//		// Subsequent calls can ignore the return value
//		DisableCacheForTest(35)
//	}
func DisableCacheForTest(minorVersion int) func() {
	return minorVersionCache.DisableForTest(minorVersion)
}

// GetMinorVersion looks up the Kubernetes minor version from the cluster.
// The version is cached and refreshed every 5 minutes, if requested.
//
// This function is safe to be called concurrently.
func GetMinorVersion() int {
	return minorVersionCache.Fetch()
}

func lookupMinorVersion(ctx context.Context, c discoverycache.Client) (int, error) {
	info, err := GetServerVersion(c)
	if err != nil {
		return 0, errors.Wrap(err, "get kubernetes server version")
	}

	minor, err := strconv.Atoi(info.Minor)

	return minor, errors.Wrap(err, "invalid kubernetes minor version")
}

func GetServerVersion(discoveryClient discovery.ServerVersionInterface) (*k8sversion.Info, error) {
	return discoveryClient.ServerVersion()
}

func GetFormattedServerVersion(discoveryClient discovery.ServerVersionInterface) (string, error) {
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"major: %s\nminor: %s\ngitVersion: %s\ngitCommit: %s\nbuildDate: %s\ngoVersion: %s\nplatform: %s\n",
		serverVersion.Major,
		serverVersion.Minor,
		serverVersion.GitVersion,
		serverVersion.GitCommit,
		serverVersion.BuildDate,
		serverVersion.GoVersion,
		serverVersion.Platform,
	), nil
}
