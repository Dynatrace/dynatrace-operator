/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package appvolumes

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"k8s.io/utils/mount"
)

func NewAppVolumePublisher(fs afero.Afero, mounter mount.Interface, db metadata.Access, path metadata.PathResolver) csivolumes.Publisher {
	return &AppVolumePublisher{
		fs:      fs,
		mounter: mounter,
		db:      db,
		path:    path,
	}
}

type AppVolumePublisher struct {
	fs      afero.Afero
	mounter mount.Interface
	db      metadata.Access
	path    metadata.PathResolver
}

func (publisher *AppVolumePublisher) PublishVolume(_ context.Context, volumeCfg csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	tenantConfig, err := publisher.db.ReadTenantConfig(metadata.TenantConfig{Name: volumeCfg.DynakubeName})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read tenant-config: "+err.Error())
	}

	hasTooManyAttempts, err := publisher.hasTooManyMountAttempts(tenantConfig, volumeCfg)
	if err != nil {
		return nil, err
	}

	if hasTooManyAttempts {
		log.Info("reached max mount attempts for pod, attaching dummy volume, monitoring disabled", "pod", volumeCfg.PodName)

		return &csi.NodePublishVolumeResponse{}, nil
	}

	if !IsArchiveAvailable(tenantConfig) {
		return nil, status.Error(
			codes.Unavailable,
			"version or digest is not yet set, csi-provisioner hasn't finished setup yet for tenant: "+tenantConfig.TenantUUID,
		)
	}

	if err := publisher.ensureMountSteps(tenantConfig, volumeCfg); err != nil {
		return nil, err
	}

	agentsVersionsMetric.WithLabelValues(tenantConfig.DownloadedCodeModuleVersion).Inc()

	return &csi.NodePublishVolumeResponse{}, nil
}

func IsArchiveAvailable(tenantConfig *metadata.TenantConfig) bool {
	return tenantConfig.DownloadedCodeModuleVersion != ""
}

func (publisher *AppVolumePublisher) UnpublishVolume(_ context.Context, volumeInfo csivolumes.VolumeInfo) (*csi.NodeUnpublishVolumeResponse, error) {
	appMount, err := publisher.db.ReadAppMount(metadata.AppMount{VolumeMetaID: volumeInfo.VolumeID})

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Info("failed to load AppMount", "error", err.Error())
	}

	if appMount == nil {
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	log.Info("loaded AppMount info", "id", appMount.VolumeMetaID, "pod name", appMount.VolumeMeta.PodName, "version", appMount.CodeModuleVersion)

	if appMount.CodeModuleVersion == "" {
		log.Info("requester has a dummy AppMount, no node-level unmount is needed")

		return &csi.NodeUnpublishVolumeResponse{}, publisher.db.DeleteAppMount(&metadata.AppMount{VolumeMetaID: appMount.VolumeMetaID})
	}

	overlayFSPath := filepath.Join(appMount.Location, dtcsi.OverlayMappedDirPath)
	publisher.unmountOneAgent(volumeInfo.TargetPath, overlayFSPath)

	if err = publisher.db.DeleteAppMount(&metadata.AppMount{VolumeMetaID: appMount.VolumeMetaID}); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info("deleted AppMount", "ID", appMount.VolumeMetaID, "PodUID", appMount.VolumeMeta.PodName, "Version", appMount.CodeModuleVersion)

	if err = publisher.fs.RemoveAll(volumeInfo.TargetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info("volume has been unpublished", "targetPath", volumeInfo.TargetPath)

	publisher.fireVolumeUnpublishedMetric(appMount.CodeModuleVersion)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (publisher *AppVolumePublisher) CanUnpublishVolume(_ context.Context, volumeInfo csivolumes.VolumeInfo) (bool, error) {
	appMount, err := publisher.db.ReadAppMount(metadata.AppMount{VolumeMetaID: volumeInfo.VolumeID})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	return appMount != nil, nil
}

func (publisher *AppVolumePublisher) fireVolumeUnpublishedMetric(volumeVersion string) {
	if len(volumeVersion) > 0 {
		agentsVersionsMetric.WithLabelValues(volumeVersion).Dec()

		var m = &dto.Metric{}
		if err := agentsVersionsMetric.WithLabelValues(volumeVersion).Write(m); err != nil {
			log.Error(err, "failed to get the value of agent version metric")
		}

		if m.GetGauge().GetValue() <= float64(0) {
			agentsVersionsMetric.DeleteLabelValues(volumeVersion)
		}
	}
}

func (publisher *AppVolumePublisher) prepareUpperDir(tenantConfig *metadata.TenantConfig, volumeCfg csivolumes.VolumeConfig) (string, error) {
	upperDir := publisher.path.OverlayVarDir(tenantConfig.TenantUUID, volumeCfg.VolumeID)
	err := publisher.fs.MkdirAll(upperDir, os.ModePerm)

	if err != nil {
		return "", errors.WithMessagef(err, "failed create overlay upper directory structure, path: %s", upperDir)
	}

	destAgentConfPath := publisher.path.OverlayVarRuxitAgentProcConf(tenantConfig.TenantUUID, volumeCfg.VolumeID)

	err = publisher.fs.MkdirAll(filepath.Dir(destAgentConfPath), os.ModePerm)
	if err != nil {
		return "", errors.WithMessagef(err, "failed create overlay upper directory agent config directory structure, path: %s", upperDir)
	}

	srcAgentConfPath := path.Join(tenantConfig.ConfigDirPath, processmoduleconfig.RuxitAgentProcPath)
	srcFile, err := publisher.fs.Open(srcAgentConfPath)

	if err != nil {
		return "", errors.WithMessagef(err, "failed to open ruxitagentproc.conf file, path: %s", srcAgentConfPath)
	}

	defer func() { _ = srcFile.Close() }()

	srcStat, err := srcFile.Stat()
	if err != nil {
		return "", errors.WithMessage(err, "failed to get source ruxitagentproc.conf file info")
	}

	destFile, err := publisher.fs.OpenFile(destAgentConfPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcStat.Mode())
	if err != nil {
		return "", errors.WithMessagef(err, "failed to open destination ruxitagentproc.conf file, path: %s", destAgentConfPath)
	}

	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return "", errors.WithMessagef(err, "failed to copy ruxitagentproc.conf file to overlay, from->to: %s -> %s", srcAgentConfPath, destAgentConfPath)
	}

	return upperDir, nil
}

func (publisher *AppVolumePublisher) mountOneAgent(tenantConfig *metadata.TenantConfig, volumeCfg csivolumes.VolumeConfig) error {
	mappedDir := publisher.path.OverlayMappedDir(tenantConfig.TenantUUID, volumeCfg.VolumeID)
	_ = publisher.fs.MkdirAll(mappedDir, os.ModePerm)

	codeModule, err := publisher.db.ReadCodeModule(metadata.CodeModule{Version: tenantConfig.DownloadedCodeModuleVersion})
	if err != nil {
		return err
	}

	upperDir, err := publisher.prepareUpperDir(tenantConfig, volumeCfg)
	if err != nil {
		return err
	}

	workDir := publisher.path.OverlayWorkDir(tenantConfig.TenantUUID, volumeCfg.VolumeID)
	_ = publisher.fs.MkdirAll(workDir, os.ModePerm)

	overlayOptions := []string{
		"lowerdir=" + codeModule.Location,
		"upperdir=" + upperDir,
		"workdir=" + workDir,
	}

	if err := publisher.fs.MkdirAll(volumeCfg.TargetPath, os.ModePerm); err != nil {
		return err
	}

	if err := publisher.mounter.Mount("overlay", mappedDir, "overlay", overlayOptions); err != nil {
		return err
	}

	if err := publisher.mounter.Mount(mappedDir, volumeCfg.TargetPath, "", []string{"bind"}); err != nil {
		_ = publisher.mounter.Unmount(mappedDir)

		return err
	}

	return nil
}

func (publisher *AppVolumePublisher) unmountOneAgent(targetPath string, overlayFSPath string) {
	if err := publisher.mounter.Unmount(targetPath); err != nil {
		log.Error(err, "Unmount failed", "path", targetPath)
	}

	if filepath.IsAbs(overlayFSPath) {
		if err := publisher.mounter.Unmount(overlayFSPath); err != nil {
			log.Error(err, "Unmount failed", "path", overlayFSPath)
		}
	}
}

func (publisher *AppVolumePublisher) ensureMountSteps(tenantConfig *metadata.TenantConfig, volumeCfg csivolumes.VolumeConfig) error {
	if err := publisher.mountOneAgent(tenantConfig, volumeCfg); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to mount oneagent volume: %s", err))
	}

	if err := publisher.storeVolume(tenantConfig, volumeCfg); err != nil {
		agentRunDirForVolume := publisher.path.AgentRunDirForVolume(tenantConfig.TenantUUID, volumeCfg.VolumeID)
		overlayFSPath := filepath.Join(agentRunDirForVolume, dtcsi.OverlayMappedDirPath)

		publisher.unmountOneAgent(volumeCfg.TargetPath, overlayFSPath)

		return status.Error(codes.Internal, fmt.Sprintf("Failed to store volume info: %s", err))
	}

	return nil
}

func (publisher *AppVolumePublisher) hasTooManyMountAttempts(tenantConfig *metadata.TenantConfig, volumeCfg csivolumes.VolumeConfig) (bool, error) {
	appMount, err := publisher.db.ReadAppMount(metadata.AppMount{VolumeMetaID: volumeCfg.VolumeID})
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		appMount, err = publisher.newAppMount(tenantConfig, volumeCfg)
		if err != nil {
			return false, err
		}

		publisher.db.CreateAppMount(appMount)
	} else if err != nil {
		return false, err
	}

	if int(appMount.MountAttempts) > int(tenantConfig.MaxFailedMountAttempts) {
		return true, nil
	}

	appMount.MountAttempts += 1

	return false, publisher.db.UpdateAppMount(appMount)
}

func (publisher *AppVolumePublisher) storeVolume(tenantConfig *metadata.TenantConfig, volumeCfg csivolumes.VolumeConfig) error {
	newAppMount, err := publisher.newAppMount(tenantConfig, volumeCfg)
	if err != nil {
		return err
	}

	log.Info("inserting AppMount", "appMount", newAppMount)

	// check if it currently exists
	appMount, err := publisher.db.ReadAppMount(metadata.AppMount{VolumeMetaID: newAppMount.VolumeMetaID})

	if appMount == nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return publisher.db.CreateAppMount(newAppMount)
	}

	if err != nil {
		return err
	}

	return publisher.db.UpdateAppMount(newAppMount)
}

func (publisher *AppVolumePublisher) newAppMount(tenantConfig *metadata.TenantConfig, volumeCfg csivolumes.VolumeConfig) (*metadata.AppMount, error) {
	codeModule, err := publisher.db.ReadCodeModule(metadata.CodeModule{Version: tenantConfig.DownloadedCodeModuleVersion})
	if err != nil {
		return nil, err
	}

	return &metadata.AppMount{
		VolumeMeta:        metadata.VolumeMeta{ID: volumeCfg.VolumeID, PodName: volumeCfg.PodName},
		CodeModule:        *codeModule,
		VolumeMetaID:      volumeCfg.VolumeID,
		CodeModuleVersion: codeModule.Version,
		Location:          publisher.path.AgentRunDirForVolume(tenantConfig.TenantUUID, volumeCfg.VolumeID),
		MountAttempts:     0,
	}, nil
}
