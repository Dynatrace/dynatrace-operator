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

package host

import (
	"context"
	"os"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/mount-utils"
)

func NewPublisher(mounter mount.Interface, path metadata.PathResolver) csivolumes.Publisher {
	return &Publisher{
		mounter: mounter,
		path:    path,
	}
}

type Publisher struct {
	mounter mount.Interface
	path    metadata.PathResolver
}

func (pub *Publisher) PublishVolume(ctx context.Context, volumeCfg *csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	if err := pub.mountStorageVolume(volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, "failed to mount osagent volume: "+err.Error())
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (pub *Publisher) mountStorageVolume(volumeCfg *csivolumes.VolumeConfig) error {
	oaStorageDir := pub.path.OsAgentDir(volumeCfg.DynakubeName)

	err := cleanupDanglingSymlink(oaStorageDir)
	if err != nil {
		log.Info("failed to cleanup dangling symlink", "path", oaStorageDir)

		return err
	}

	err = os.MkdirAll(oaStorageDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	if err := os.MkdirAll(volumeCfg.TargetPath, os.ModePerm); err != nil {
		log.Info("failed to create directory for osagent-storage mount", "directory", oaStorageDir)

		return err
	}

	if err := pub.mounter.Mount(oaStorageDir, volumeCfg.TargetPath, "", []string{"bind"}); err != nil {
		_ = pub.mounter.Unmount(oaStorageDir)

		log.Info("failed to mount directory for osagent-storage mount", "directory", oaStorageDir)

		return err
	}

	return nil
}

func cleanupDanglingSymlink(hostDir string) error {
	linkInfo, err := os.Lstat(hostDir)
	if err == nil && linkInfo.Mode()&os.ModeSymlink != 0 {
		_, err := os.Stat(hostDir)
		if os.IsNotExist(err) {
			log.Debug("found dangling symlink", "path", hostDir)

			return os.Remove(hostDir)
		}
	}

	return nil
}
