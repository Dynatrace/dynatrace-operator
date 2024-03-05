package metadata

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"gorm.io/gorm"
)

func dataMigration(tx *gorm.DB) error {
	var dynakubes []Dynakube

	tx.Table("dynakubes").Find(&dynakubes)

	for _, d := range dynakubes {
		tc := TenantConfig{
			Name:                        d.Name,
			TenantUUID:                  d.TenantUUID,
			ConfigDirPath:               filepath.Join(filepath.Join(dtcsi.DataPath, d.TenantUUID), d.Name, dtcsi.SharedAgentConfigDir),
			DownloadedCodeModuleVersion: d.LatestVersion,
			MaxFailedMountAttempts:      int64(d.MaxFailedMountAttempts),
		}

		result := tx.Create(&tc)
		if result.Error != nil {
			return result.Error
		}

		cm := CodeModule{
			Version:  d.LatestVersion,
			Location: filepath.Join(dtcsi.DataPath, dtcsi.SharedAgentBinDir),
		}
		tx.Create(&cm)
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
		tx.Create(&vm)

		am := AppMount{
			CodeModuleVersion: v.Version,
			VolumeMetaID:      vm.ID,
			Location:          filepath.Join(filepath.Join(filepath.Join(dtcsi.DataPath, v.TenantUUID), dtcsi.AgentRunDir), vm.ID),
			MountAttempts:     int64(v.MountAttempts),
		}
		tx.Create(&am)
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
			MountAttempts: mountAttempts,
		}
		tx.Create(&om)
	}

	return nil
}
