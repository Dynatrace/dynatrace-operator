package metadata

import (
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"gorm.io/gorm"
)

func migrateDynakubes(tx *gorm.DB) error {
	var dynakubes []Dynakube

	tx.Table("dynakubes").Find(&dynakubes)

	pr := PathResolver{RootDir: dtcsi.DataPath}

	for _, d := range dynakubes {
		var version string
		if d.LatestVersion != "" {
			version = d.LatestVersion
		} else if d.ImageDigest != "" {
			version = d.ImageDigest
		}

		tc := TenantConfig{
			Name:                        d.Name,
			TenantUUID:                  d.TenantUUID,
			ConfigDirPath:               pr.AgentConfigDir(d.TenantUUID, d.Name),
			DownloadedCodeModuleVersion: version,
			MaxFailedMountAttempts:      int64(d.MaxFailedMountAttempts),
		}

		result := tx.Create(&tc)
		if result.Error != nil {
			log.Error(result.Error, "failed to create TenantConfig")
		}

		cm := CodeModule{
			Version:  version,
			Location: pr.AgentSharedBinaryDirForAgent(version),
		}
		tx.Create(&cm)

		if result.Error != nil {
			log.Error(result.Error, "failed to create CodeModule")
		}
	}

	return nil
}

func migrateVolumes(tx *gorm.DB) error {
	// Old `Volumes` tables is where we store the information about the mounted volumes that are used
	// for Application monitoring (codemodules)
	// the reason the names is so generic is that originally that was the only kind of volume
	//
	// New `AppMount` table is where we store information that is ONLY relevant for volumes that
	// are for Application monitoring.
	var volumes []Volume

	pr := PathResolver{RootDir: dtcsi.DataPath}

	tx.Table("volumes").Find(&volumes)

	for _, v := range volumes {
		vm := VolumeMeta{
			ID:           v.VolumeID,
			PodUid:       "",
			PodName:      v.PodName,
			PodNamespace: "",
		}

		result := tx.Create(&vm)
		if result.Error != nil {
			log.Error(result.Error, "failed to create VolumeMeta")
		}

		am := AppMount{
			CodeModuleVersion: v.Version,
			VolumeMetaID:      vm.ID,
			Location:          pr.AgentRunDirForVolume(v.TenantUUID, vm.ID),
			MountAttempts:     int64(v.MountAttempts),
		}

		result = tx.Create(&am)
		if result.Error != nil {
			log.Error(result.Error, "failed to create AppMount")
		}
	}

	return nil
}

func migrateOsAgentVolumes(tx *gorm.DB) error {
	// Old `OsAgentVolume` table is where we store the information about the mounted volumes that
	// are used for the OsAgent this was just bolted on,
	// because they need to be handled differently (and was not properly finished)
	//
	// New `OsMount` table is where we store information that is ONLY relevant for volumes that are
	// for the OsAgent.
	var osAgentVolumnes []OsAgentVolume

	pr := PathResolver{RootDir: dtcsi.DataPath}

	tx.Table("osagent_volumes").Find(&osAgentVolumnes)

	for _, ov := range osAgentVolumnes {
		if !ov.Mounted {
			continue
		}

		vm := VolumeMeta{
			ID: ov.VolumeID,
		}

		result := tx.Create(&vm)
		if result.Error != nil {
			log.Error(result.Error, "failed to create VolumeMeta")
		}

		var mountAttempts int64
		if ov.Mounted {
			mountAttempts = 1
		}

		// This is a workaround for not having enough information in the current database tables to migrate 100% correctly.
		// This is fine as we don't currently use this information for anything.
		tc := TenantConfig{TenantUUID: ov.TenantUUID}
		tx.First(&tc)

		om := OSMount{
			TenantConfigUID: tc.UID,
			TenantUUID:      ov.TenantUUID,
			VolumeMetaID:    vm.ID,
			Location:        pr.AgentRunDirForVolume(ov.TenantUUID, ov.VolumeID),
			MountAttempts:   mountAttempts,
		}
		result = tx.Create(&om)

		if result.Error != nil {
			log.Error(result.Error, "failed to create OSMount")
		}
	}

	return nil
}

func dataMigration(tx *gorm.DB) error {
	if err := migrateDynakubes(tx); err != nil {
		return err
	}

	if err := migrateVolumes(tx); err != nil {
		return err
	}

	return migrateOsAgentVolumes(tx)
}
