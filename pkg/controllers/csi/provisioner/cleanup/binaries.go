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

func (c *Cleaner) removeUnusedBinaries(ctx context.Context, dks []*dynakube.DynaKube, fsState fsState) {
	c.removeOldBinarySymlinks(ctx, dks, fsState)

	keptBins, err := c.collectStillMountedBins(ctx)
	if err != nil {
		return
	}

	relevantLatestBins := c.collectRelevantLatestBins(ctx, dks)

	maps.Copy(keptBins, relevantLatestBins)

	c.removeOldSharedBinaries(ctx, keptBins)
}

func (c *Cleaner) removeOldSharedBinaries(ctx context.Context, keptBins map[string]bool) {
	log := logd.FromContext(ctx)

	sharedBins, err := os.ReadDir(c.path.AgentSharedBinaryDirBase())
	if err != nil {
		log.Info("failed to list the shared binaries directory, skipping unused binaries cleanup")

		return
	}

	for _, dir := range sharedBins {
		sharedBinPath := c.path.AgentSharedBinaryDirForAgent(dir.Name())

		_, ok := keptBins[sharedBinPath]
		if !ok {
			err := os.RemoveAll(sharedBinPath)
			if err != nil {
				log.Error(err, "failed to remove shared binary", "path", sharedBinPath)

				continue
			}

			log.Info("removed old shared binary", "path", sharedBinPath)
		}
	}
}

func (c *Cleaner) removeOldBinarySymlinks(ctx context.Context, dks []*dynakube.DynaKube, fsState fsState) {
	log := logd.FromContext(ctx)

	shouldBePresent := map[string]bool{}
	for _, dk := range dks {
		shouldBePresent[dk.Name] = true
	}

	for _, dkDir := range fsState.binDks {
		if _, ok := shouldBePresent[dkDir]; !ok {
			latest := c.path.LatestAgentBinaryForDynaKube(dkDir)
			if err := os.Remove(latest); err == nil {
				log.Info("removed old latest bin symlink", "path", latest)
			}
		}
	}

	for _, depDir := range fsState.deprecatedDks {
		if _, ok := shouldBePresent[depDir]; !ok { // for the rare case where dk.Name == tenantUUID
			latest := c.path.LatestAgentBinaryForDynaKube(depDir)
			if err := os.Remove(latest); err == nil {
				log.Info("removed old deprecated latest bin symlink", "path", latest)
			}
		}
	}
}

func (c *Cleaner) collectStillMountedBins(ctx context.Context) (map[string]bool, error) {
	log := logd.FromContext(ctx)

	mountedBins := map[string]bool{}

	// App mounts live at the kubelet target path, not under the CSI root, so they can only be
	// found via their lower directory, which points at the code module they are using.
	log.Debug("looking for code modules that are still mounted", "dir", c.path.AgentSharedBinaryDirBase())

	overlays, err := metadata.GetOverlayMountsWithLowerDirIn(ctx, c.mounter, c.path.AgentSharedBinaryDirBase())
	if err != nil {
		log.Info("failed to list active overlay mounts, skipping unused binaries cleanup")

		return nil, err
	}

	for _, overlay := range overlays {
		for _, lowerDir := range overlay.LowerDirs {
			mountedBins[lowerDir] = true
		}
	}

	if len(mountedBins) > 0 {
		log.Info("binaries to keep because they are still mounted", "paths", strings.Join(slices.Collect(maps.Keys(mountedBins)), ","))
	}

	return mountedBins, nil
}

func (c *Cleaner) collectRelevantLatestBins(ctx context.Context, dks []*dynakube.DynaKube) map[string]bool {
	log := logd.FromContext(ctx)

	latestBins := map[string]bool{}

	for _, dk := range dks {
		if !dk.OneAgent().IsAppInjectionNeeded() {
			continue
		}

		latestLink := c.path.LatestAgentBinaryForDynaKube(dk.Name)

		c.addRelevantPath(ctx, latestLink, latestBins)
	}

	if len(latestBins) > 0 {
		log.Info("binaries to keep because they are the latest for existing dynakubes", "paths", strings.Join(slices.Collect(maps.Keys(latestBins)), ","))
	}

	return latestBins
}
