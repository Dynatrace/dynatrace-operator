// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package cleanup

import (
	"context"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

func (c *Cleaner) isMountPoint(file string) (bool, error) {
	isMountPoint, err := c.mounter.IsMountPoint(file)
	if os.IsNotExist(err) {
		// if the file is a symlink, what it points to may also not exist
		return false, nil
	}

	return isMountPoint, err
}

func (c *Cleaner) removeHostMounts(ctx context.Context, dks []dynakube.DynaKube, fsState fsState) {
	log := logd.FromContext(ctx)

	relevantHostDirs := c.collectRelevantHostDirs(ctx, dks)

	for _, hostDK := range fsState.hostDks {
		possibleHostDirs := []string{
			c.path.OSAgentDir(hostDK),
			c.path.OldOSAgentDir(hostDK),
		}

		for _, hostDir := range possibleHostDirs {
			_, err := os.Stat(hostDir)
			if os.IsNotExist(err) {
				log.Debug("host dir path doesn't exist, moving to the next one", "path", hostDir)

				continue
			} else if err != nil {
				log.Debug("failed to determine stat of host dir path, moving to the next one", "path", hostDir, "err", err)

				continue
			}

			isMountPoint, err := c.isMountPoint(hostDir)
			if err == nil && !isMountPoint && !relevantHostDirs[hostDir] {
				err = os.RemoveAll(hostDir)
				if err == nil {
					log.Info("removed old host mount directory", "path", hostDir)
				}
			}
		}
	}
}

func (c *Cleaner) collectRelevantHostDirs(ctx context.Context, dks []dynakube.DynaKube) map[string]bool {
	log := logd.FromContext(ctx)

	hostDirs := map[string]bool{}

	for _, dk := range dks {
		if !dk.OneAgent().IsReadOnlyFSSupported() {
			continue
		}

		hostDir := c.path.OSAgentDir(dk.Name)

		hostDirs[hostDir] = true

		c.safeAddRelevantPath(ctx, hostDir, hostDirs)

		tenantUUID, err := metadata.TenantUUIDFromAPIURL(dk.APIURL())
		if err != nil {
			log.Error(err, "malformed APIURL for dynakube during host mount directory cleanup", "dynakube", dk.Name, "apiUrl", dk.APIURL())

			continue
		}

		deprecatedHostDirLink := c.path.OldOSAgentDir(tenantUUID)
		c.safeAddRelevantPath(ctx, deprecatedHostDirLink, hostDirs)
	}

	if len(hostDirs) > 0 {
		log.Info("host directories to keep because they have a related dynakube", "paths", strings.Join(slices.Collect(maps.Keys(hostDirs)), ","))
	}

	return hostDirs
}
