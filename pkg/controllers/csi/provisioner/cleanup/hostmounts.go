package cleanup

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"golang.org/x/exp/maps"
	"k8s.io/mount-utils"
)

func (c *Cleaner) isMountPoint(file string) (bool, error) {
	fakeMounter, ok := c.mounter.(*mount.FakeMounter)
	if ok {
		// you can't use the fake mounter IsLikelyNotMountPoint, as it will still use the os package
		err, ok := fakeMounter.MountCheckErrors[file]
		if ok {
			if err == nil {
				return true, nil
			}

			return false, err
		} else {
			return false, nil
		}
	}

	return c.mounter.IsMountPoint(file)
}

func (c *Cleaner) removeHostMounts(dks []dynakube.DynaKube, fsState fsState) {
	relevantHostDirs := c.collectRelevantHostDirs(dks)

	for _, hostDk := range fsState.hostDks {
		possibleHostDirs := []string{
			c.path.OsAgentDir(hostDk),
			c.path.OldOsAgentDir(hostDk),
		}

		for _, hostDir := range possibleHostDirs {
			isMountPoint, err := c.isMountPoint(hostDir)
			if err == nil && !isMountPoint && !relevantHostDirs[hostDir] {
				err := c.fs.RemoveAll(hostDir)
				if err == nil {
					log.Info("removed old host mount directory", "path", hostDir)
				}
			}
		}
	}
}

func (c *Cleaner) collectRelevantHostDirs(dks []dynakube.DynaKube) map[string]bool {
	hostDirs := map[string]bool{}

	for _, dk := range dks {
		if !dk.OneAgent().IsOneAgentModeSupportingReadOnlyFS() {
			continue
		}

		hostDir := c.path.OsAgentDir(dk.Name)

		hostDirs[hostDir] = true

		c.safeAddRelevantPath(hostDir, hostDirs)

		tenantUUID, err := metadata.TenantUUIDFromApiUrl(dk.ApiUrl())
		if err != nil {
			log.Error(err, "malformed ApiUrl for dynakube during host mount directory cleanup", "dk", dk.Name, "apiUrl", dk.ApiUrl())

			continue
		}

		deprecatedHostDirLink := c.path.OldOsAgentDir(tenantUUID)
		c.safeAddRelevantPath(deprecatedHostDirLink, hostDirs)
	}

	if len(hostDirs) > 0 {
		log.Info("host directories to keep because they have a related dynakube", "paths", strings.Join(maps.Keys(hostDirs), ","))
	}

	return hostDirs
}
