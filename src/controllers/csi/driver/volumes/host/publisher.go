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

	csivolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const oneAgentHostDir = "host"

func NewHostVolumePublisher(client client.Client, fs afero.Afero, mounter mount.Interface, db metadata.Access, path metadata.PathResolver) csivolumes.Publisher {
	return &HostVolumePublisher{
		client:  client,
		fs:      fs,
		mounter: mounter,
		db:      db,
		path:    path,
	}
}

type HostVolumePublisher struct {
	client  client.Client
	fs      afero.Afero
	mounter mount.Interface
	db      metadata.Access
	path    metadata.PathResolver
}

func (publisher *HostVolumePublisher) PublishVolume(ctx context.Context, volumeCfg *csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	if err := publisher.mountOneAgent(volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount host agent volume: %s", err.Error()))
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

func (publisher *HostVolumePublisher) UnpublishVolume(_ context.Context, volumeInfo *csivolumes.VolumeInfo) (*csi.NodeUnpublishVolumeResponse, error) {
	hostDir := publisher.path.EnvDir(oneAgentHostDir)

	if err := publisher.umountOneAgent(volumeInfo.TargetPath, hostDir); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount host agent volume: %s", err.Error()))
	}

	if err := publisher.fs.RemoveAll(volumeInfo.TargetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info("host agent volume has been unpublished", "targetPath", volumeInfo.TargetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (publisher *HostVolumePublisher) mountOneAgent(volumeCfg *csivolumes.VolumeConfig) error {
	hostDir := publisher.path.EnvDir(oneAgentHostDir)
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

func (publisher *HostVolumePublisher) umountOneAgent(targetPath string, hostDir string) error {
	if err := publisher.mounter.Unmount(targetPath); err != nil {
		log.Error(err, "Unmount failed", "path", targetPath)
	}

	return nil
}
