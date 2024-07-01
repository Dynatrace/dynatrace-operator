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

package hostvolumes

import (
	"context"
	"fmt"
	"os"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"k8s.io/utils/mount"
)

const failedToGetOsAgentVolumePrefix = "failed to get OSMount from database: "

func NewHostVolumePublisher(fs afero.Afero, mounter mount.Interface, db metadata.Access, path metadata.PathResolver) csivolumes.Publisher {
	return &HostVolumePublisher{
		fs:      fs,
		mounter: mounter,
		db:      db,
		path:    path,
	}
}

type HostVolumePublisher struct {
	fs      afero.Afero
	mounter mount.Interface
	db      metadata.Access
	path    metadata.PathResolver
}

func (publisher *HostVolumePublisher) PublishVolume(ctx context.Context, volumeCfg csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	tenantConfig, err := publisher.db.ReadTenantConfig(metadata.TenantConfig{Name: volumeCfg.DynakubeName})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read tenant-config: "+err.Error())
	}

	osMount, err := publisher.db.ReadUnscopedOSMount(metadata.OSMount{TenantUUID: tenantConfig.TenantUUID})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.Internal, failedToGetOsAgentVolumePrefix+err.Error())
	}

	if osMount != nil && osMount.DeletedAt.Valid {
		osMount.VolumeMeta = metadata.VolumeMeta{
			ID:      volumeCfg.VolumeID,
			PodName: volumeCfg.PodName,
		}
		osMount.TenantConfig = *tenantConfig

		if err := publisher.mountOneAgent(osMount, volumeCfg); err != nil {
			return nil, status.Error(codes.Internal, "failed to mount OSMount: "+err.Error())
		}

		osMount, err = publisher.db.RestoreOSMount(osMount)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to restore OSMount: "+err.Error())
		}
	}

	if osMount == nil && errors.Is(err, gorm.ErrRecordNotFound) {
		osMount := metadata.OSMount{
			VolumeMeta:    metadata.VolumeMeta{ID: volumeCfg.VolumeID, PodName: volumeCfg.PodName},
			VolumeMetaID:  volumeCfg.VolumeID,
			TenantUUID:    tenantConfig.TenantUUID,
			Location:      publisher.path.OsAgentDir(tenantConfig.TenantUUID),
			MountAttempts: 0,
			TenantConfig:  *tenantConfig,
		}

		if err := publisher.mountOneAgent(&osMount, volumeCfg); err != nil {
			return nil, status.Error(codes.Internal, "failed to mount OSMount: "+err.Error())
		}

		if err := publisher.db.CreateOSMount(&osMount); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to insert osmount to database. info: %v err: %s", osMount, err.Error()))
		}
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (publisher *HostVolumePublisher) UnpublishVolume(ctx context.Context, volumeInfo csivolumes.VolumeInfo) (*csi.NodeUnpublishVolumeResponse, error) {
	osMount, err := publisher.db.ReadOSMount(metadata.OSMount{VolumeMetaID: volumeInfo.VolumeID})

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	if err != nil {
		return nil, status.Error(codes.Internal, failedToGetOsAgentVolumePrefix+err.Error())
	}

	if osMount == nil {
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	publisher.unmountOneAgent(volumeInfo.TargetPath)

	if err := publisher.db.DeleteOSMount(&metadata.OSMount{TenantUUID: osMount.TenantUUID}); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update OSMount to database. info: %v err: %s", osMount, err.Error()))
	}

	log.Info("OSMount has been unpublished", "targetPath", volumeInfo.TargetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (publisher *HostVolumePublisher) CanUnpublishVolume(ctx context.Context, volumeInfo csivolumes.VolumeInfo) (bool, error) {
	volume, err := publisher.db.ReadOSMount(metadata.OSMount{VolumeMetaID: volumeInfo.VolumeID})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, status.Error(codes.Internal, failedToGetOsAgentVolumePrefix+err.Error())
	}

	return volume != nil, nil
}

func (publisher *HostVolumePublisher) mountOneAgent(osMount *metadata.OSMount, volumeCfg csivolumes.VolumeConfig) error {
	hostDir := osMount.Location
	_ = publisher.fs.MkdirAll(hostDir, os.ModePerm)

	if err := publisher.fs.MkdirAll(volumeCfg.TargetPath, os.ModePerm); err != nil {
		return err
	}

	if err := publisher.mounter.Mount(hostDir, volumeCfg.TargetPath, "", []string{"bind"}); err != nil {
		_ = publisher.mounter.Unmount(hostDir)

		return err
	}

	return nil
}

func (publisher *HostVolumePublisher) unmountOneAgent(targetPath string) {
	if err := publisher.mounter.Unmount(targetPath); err != nil {
		log.Error(err, "Unmount failed", "path", targetPath)
	}
}
