package cleanup

func (c *Cleaner) removeDeprecatedMounts(fsState fsState) {
	stillMountedCounter := 0

	for _, depDir := range fsState.deprecatedDks {
		runDir := c.path.AgentRunDir(depDir)

		volumeDirs, err := c.fs.ReadDir(runDir)
		if err != nil {
			log.Info("couldn't list volume dirs", "path", runDir)

			continue
		}

		for _, volumeDir := range volumeDirs {
			mappedDir := c.path.OverlayMappedDir(depDir, volumeDir.Name())

			isEmpty, _ := c.fs.IsEmpty(mappedDir)
			if isEmpty {
				volumeDirPath := c.path.AgentRunDirForVolume(depDir, volumeDir.Name())

				err := c.fs.RemoveAll(volumeDirPath)
				if err == nil {
					log.Info("removed unused volume", "path", volumeDirPath)

					continue
				}
			}

			stillMountedCounter++
		}

		isEmpty, _ := c.fs.IsEmpty(runDir)
		if !isEmpty {
			continue
		}

		err = c.fs.RemoveAll(runDir)
		if err != nil {
			log.Info("removed empty deprecated run dir", "path", runDir)

			continue
		}

		tenantDir := c.path.DynaKubeDir(depDir)

		err = c.fs.RemoveAll(c.path.DynaKubeDir(tenantDir))
		if err != nil {
			log.Info("removed empty deprecated dir", "path", tenantDir)
		}
	}

	if stillMountedCounter > 0 {
		log.Info("there are a still mounted deprecated app mounts", "count", stillMountedCounter)
	}
}
