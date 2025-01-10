package cleanup

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"golang.org/x/exp/maps"
)

func (c Cleaner) removeHostMounts(dks []dynakube.DynaKube, fsState fsState) {
	relevantHostDirs := c.collectRelevantHostDirs(dks)

	for _, hostDk := range fsState.hostDks {
		hostDir := c.path.OsAgentDir(hostDk)

		isMountPoint, err := c.mounter.IsMountPoint(hostDir)
		if err == nil && !isMountPoint && !relevantHostDirs[hostDir] {
			err := c.fs.RemoveAll(hostDir)
			if err == nil {
				log.Info("removed old host mount directory", "path", hostDir)
			}
		}
	}
}

func (c Cleaner) collectRelevantHostDirs(dks []dynakube.DynaKube) map[string]bool {
	hostDirs := map[string]bool{}

	for _, dk := range dks {
		if !dk.NeedsOneAgent() {
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

		deprecatedHostDirLink := c.path.OsAgentDir(tenantUUID)
		c.safeAddRelevantPath(deprecatedHostDirLink, hostDirs)
	}

	if len(hostDirs) > 0 {
		log.Info("host directories to keep because they have a related dynakube", "paths", strings.Join(maps.Keys(hostDirs), ","))
	}

	return hostDirs
}
