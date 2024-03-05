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
	"time"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
)

const failedToGetOsAgentVolumePrefix = "failed to get osagent volume info from database: "

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

func (publisher *HostVolumePublisher) PublishVolume(ctx context.Context, volumeCfg *csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	bindCfg, err := csivolumes.NewBindConfig(ctx, publisher.db, volumeCfg)
	if err != nil {
		return nil, err
	}

	if err := publisher.mountOneAgent(bindCfg.TenantUUID, volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, "failed to mount osagent volume: "+err.Error())
	}

	volume, err := publisher.db.GetOsAgentVolumeViaTenantUUID(ctx, bindCfg.TenantUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, failedToGetOsAgentVolumePrefix+err.Error())
	}

	timestamp := time.Now()
	if volume == nil {
		storage := metadata.OsAgentVolume{
			VolumeID:     volumeCfg.VolumeID,
			TenantUUID:   bindCfg.TenantUUID,
			Mounted:      true,
			LastModified: &timestamp,
		}
		if err := publisher.db.InsertOsAgentVolume(ctx, &storage); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to insert osagent volume info to database. info: %v err: %s", storage, err.Error()))
		}
	} else {
		volume.VolumeID = volumeCfg.VolumeID
		volume.Mounted = true
		volume.LastModified = &timestamp

		if err := publisher.db.UpdateOsAgentVolume(ctx, volume); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update osagent volume info to database. info: %v err: %s", volume, err.Error()))
		}
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (publisher *HostVolumePublisher) UnpublishVolume(ctx context.Context, volumeInfo *csivolumes.VolumeInfo) (*csi.NodeUnpublishVolumeResponse, error) {
	volume, err := publisher.db.GetOsAgentVolumeViaVolumeID(ctx, volumeInfo.VolumeID)
	if err != nil {
		return nil, status.Error(codes.Internal, failedToGetOsAgentVolumePrefix+err.Error())
	}

	if volume == nil {
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	publisher.umountOneAgent(volumeInfo.TargetPath)

	timestamp := time.Now()
	volume.Mounted = false
	volume.LastModified = &timestamp

	if err := publisher.db.UpdateOsAgentVolume(ctx, volume); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update osagent volume info to database. info: %v err: %s", volume, err.Error()))
	}

	log.Info("osagent volume has been unpublished", "targetPath", volumeInfo.TargetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (publisher *HostVolumePublisher) CanUnpublishVolume(ctx context.Context, volumeInfo *csivolumes.VolumeInfo) (bool, error) {
	volume, err := publisher.db.GetOsAgentVolumeViaVolumeID(ctx, volumeInfo.VolumeID)
	if err != nil {
		return false, status.Error(codes.Internal, failedToGetOsAgentVolumePrefix+err.Error())
	}

	return volume != nil, nil
}

func (publisher *HostVolumePublisher) mountOneAgent(tenantUUID string, volumeCfg *csivolumes.VolumeConfig) error {
	hostDir := publisher.path.OsAgentDir(tenantUUID)
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

func (publisher *HostVolumePublisher) umountOneAgent(targetPath string) {
	if err := publisher.mounter.Unmount(targetPath); err != nil {
		log.Error(err, "Unmount failed", "path", targetPath)
	}
}
