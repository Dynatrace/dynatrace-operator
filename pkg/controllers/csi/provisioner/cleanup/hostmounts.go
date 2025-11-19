package cleanup

import (
	"os"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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

	isMountPoint, err := c.mounter.IsMountPoint(file)
	if os.IsNotExist(err) {
		// this is a different not exist err from the previous,
		// if the file is a symlink, then what the symlink is pointing to can also not exist
		// and IsMountPoint follows symlink without question
		return false, nil
	}

	return isMountPoint, err
}

func (c *Cleaner) removeHostMounts(dks []dynakube.DynaKube, fsState fsState) {
	relevantHostDirs := c.collectRelevantHostDirs(dks)

	for _, hostDk := range fsState.hostDks {
		possibleHostDirs := []string{
			c.path.OsAgentDir(hostDk),
			c.path.OldOsAgentDir(hostDk),
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

func (c *Cleaner) collectRelevantHostDirs(dks []dynakube.DynaKube) map[string]bool {
	hostDirs := map[string]bool{}

	for _, dk := range dks {
		if !dk.OneAgent().IsReadOnlyFSSupported() {
			continue
		}

		hostDir := c.path.OsAgentDir(dk.Name)

		hostDirs[hostDir] = true

		c.safeAddRelevantPath(hostDir, hostDirs)

		tenantUUID, err := metadata.TenantUUIDFromAPIURL(dk.APIURL())
		if err != nil {
			log.Error(err, "malformed APIURL for dynakube during host mount directory cleanup", "dk", dk.Name, "apiUrl", dk.APIURL())

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
