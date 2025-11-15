package cleanup

import "os"

func (c *Cleaner) removeDeprecatedMounts(fsState fsState) {
	stillMountedCounter := 0

	for _, depDir := range fsState.deprecatedDks {
		runDir := c.path.AgentRunDir(depDir)

		volumeDirs, err := os.ReadDir(runDir)
		if err != nil {
			log.Info("couldn't list volume dirs", "path", runDir)

			continue
		}

		for _, volumeDir := range volumeDirs {
			mappedDir := c.path.OverlayMappedDir(depDir, volumeDir.Name())

			subDirs, _ := os.ReadDir(mappedDir)
			if len(subDirs) == 0 {
				volumeDirPath := c.path.AgentRunDirForVolume(depDir, volumeDir.Name())

				err := os.RemoveAll(volumeDirPath)
				if err == nil {
					log.Info("removed unused volume", "path", volumeDirPath)

					continue
				}
			}

			stillMountedCounter++
		}

		subDirs, _ := os.ReadDir(runDir)
		if len(subDirs) > 0 {
			continue
		}

		err = os.RemoveAll(runDir)
		if err == nil {
			log.Info("removed empty deprecated run dir", "path", runDir)
		} else {
			continue
		}

		tenantDir := c.path.TenantDir(depDir)

		err = os.RemoveAll(c.path.DynaKubeDir(tenantDir))
		if err == nil {
			log.Info("removed empty deprecated dir", "path", tenantDir)
		}
	}

	if stillMountedCounter > 0 {
		log.Info("there are a still mounted deprecated app mounts", "count", stillMountedCounter)
	}
}
