package metadata

import (
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"gorm.io/gorm"
)

func dataMigration(tx *gorm.DB) error {
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

	var volumes []Volume

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

	var osAgentVolumnes []OsAgentVolume

	tx.Table("osagent_volumes").Find(&osAgentVolumnes)

	for _, ov := range osAgentVolumnes {
		var mountAttempts int64
		if ov.Mounted {
			mountAttempts = 1
		}

		om := OSMount{
			TenantUUID:    ov.TenantUUID,
			VolumeMetaID:  ov.VolumeID,
			Location:      pr.AgentRunDirForVolume(ov.TenantUUID, ov.VolumeID),
			MountAttempts: mountAttempts,
		}
		tx.Create(&om)
	}

	return nil
}
