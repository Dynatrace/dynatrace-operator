package cleanup

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"golang.org/x/exp/maps"
)

func (c *Cleaner) removeUnusedBinaries(dks []dynakube.DynaKube, fsState fsState) {
	c.removeOldBinarySymlinks(dks, fsState)

	keptBins, err := c.collectStillMountedBins()
	if err != nil {
		return
	}

	relevantLatestBins := c.collectRelevantLatestBins(dks)

	for k, v := range relevantLatestBins {
		keptBins[k] = v
	}

	c.removeOldSharedBinaries(keptBins)
}

func (c *Cleaner) removeOldSharedBinaries(keptBins map[string]bool) {
	sharedBins, err := c.fs.ReadDir(c.path.AgentSharedBinaryDirBase())
	if err != nil {
		log.Info("failed to list the shared binaries directory, skipping unused binaries cleanup")

		return
	}

	for _, dir := range sharedBins {
		sharedBinPath := c.path.AgentSharedBinaryDirForAgent(dir.Name())

		_, ok := keptBins[sharedBinPath]
		if !ok {
			err := c.fs.RemoveAll(sharedBinPath)
			if err != nil {
				log.Error(err, "failed to remove shared binary", "path", sharedBinPath)

				continue
			}

			log.Info("removed old shared binary", "path", sharedBinPath)
		}
	}
}

func (c *Cleaner) removeOldBinarySymlinks(dks []dynakube.DynaKube, fsState fsState) {
	shouldBePresent := map[string]bool{}
	for _, dk := range dks {
		shouldBePresent[dk.Name] = true
	}

	for _, dkDir := range fsState.binDks {
		if _, ok := shouldBePresent[dkDir]; !ok {
			latest := c.path.LatestAgentBinaryForDynaKube(dkDir)
			if err := c.fs.Remove(latest); err == nil {
				log.Info("removed old latest bin symlink", "path", latest)
			}
		}
	}

	for _, depDir := range fsState.deprecatedDks {
		if _, ok := shouldBePresent[depDir]; !ok { // for the rare case where dk.Name == tenantUUID
			latest := c.path.LatestAgentBinaryForDynaKube(depDir)
			if err := c.fs.Remove(latest); err == nil {
				log.Info("removed old deprecated latest bin symlink", "path", latest)
			}
		}
	}
}

func (c *Cleaner) collectStillMountedBins() (map[string]bool, error) {
	mountedBins := map[string]bool{}

	overlays, err := metadata.GetRelevantOverlayMounts(c.mounter, c.path.RootDir)
	if err != nil {
		log.Info("failed to list active overlay mounts, skipping unused binaries cleanup")

		return nil, err
	}

	for _, overlay := range overlays {
		mountedBins[overlay.LowerDir] = true
	}

	if len(mountedBins) > 0 {
		log.Info("binaries to keep because they are still mounted", "paths", strings.Join(maps.Keys(mountedBins), ","))
	}

	return mountedBins, nil
}

func (c *Cleaner) collectRelevantLatestBins(dks []dynakube.DynaKube) map[string]bool {
	latestBins := map[string]bool{}

	for _, dk := range dks {
		if !dk.OneAgent().IsAppInjectionNeeded() {
			continue
		}

		latestLink := c.path.LatestAgentBinaryForDynaKube(dk.Name)

		c.addRelevantPath(latestLink, latestBins)
	}

	if len(latestBins) > 0 {
		log.Info("binaries to keep because they are the latest for existing dynakubes", "paths", strings.Join(maps.Keys(latestBins), ","))
	}

	return latestBins
}
