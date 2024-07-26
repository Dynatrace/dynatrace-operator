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
	goerrors "errors"
	"os"
	"strings"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
)

func NewHostVolumePublisher(fs afero.Afero, mounter mount.Interface, path metadata.PathResolver) csivolumes.Publisher {
	return &HostVolumePublisher{
		fs:           fs,
		mounter:      mounter,
		path:         path,
		isNotMounted: mount.IsNotMountPoint,
	}
}

// necessary for mocking, as the MounterMock will use the os package
type mountChecker func(mounter mount.Interface, file string) (bool, error)

type HostVolumePublisher struct {
	fs           afero.Afero
	mounter      mount.Interface
	isNotMounted mountChecker
	path         metadata.PathResolver
}

func (publisher *HostVolumePublisher) PublishVolume(ctx context.Context, volumeCfg csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	hostDir := publisher.path.OsMountDir()
	_ = publisher.fs.MkdirAll(hostDir, os.ModePerm)
	// If the OSAgents were removed forcefully, we might not get the unmount request, so we can't fully relay on the database, and have to directly check if its mounted or not
	isNotMounted, err := publisher.isNotMounted(publisher.mounter, hostDir)
	if err != nil {
		return nil, err
	}

	if !isNotMounted {
		return &csi.NodePublishVolumeResponse{}, goerrors.New("previous OSMount is yet to be unmounted, there can be only 1 OSMount per tenant per node, blocking until unmount") // don't want to have the stacktrace here, it just pollutes the logs
	}

	if err := publisher.mountOneAgent(volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, "failed to mount OSMount: "+err.Error())
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (publisher *HostVolumePublisher) UnpublishVolume(ctx context.Context, volumeInfo csivolumes.VolumeInfo) (*csi.NodeUnpublishVolumeResponse, error) {
	publisher.unmountOneAgent(volumeInfo.TargetPath)

	log.Info("OSMount has been unpublished", "targetPath", volumeInfo.TargetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (publisher *HostVolumePublisher) CanUnpublishVolume(ctx context.Context, volumeInfo csivolumes.VolumeInfo) (bool, error) { // probably not even needed
	return strings.Contains(volumeInfo.TargetPath, "osagent-storage"), nil // TODO: fix to use constant, fix import cycle
}

func (publisher *HostVolumePublisher) mountOneAgent(volumeCfg csivolumes.VolumeConfig) error {
	if err := publisher.fs.MkdirAll(volumeCfg.TargetPath, os.ModePerm); err != nil {
		return err
	}

	hostDir := publisher.path.OsMountDir()
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
