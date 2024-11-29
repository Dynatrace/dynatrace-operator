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
	"os"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/mount-utils"
)

func NewPublisher(fs afero.Afero, mounter mount.Interface, path metadata.PathResolver) csivolumes.Publisher {
	return &Publisher{
		fs:      fs,
		mounter: mounter,
		path:    path,
	}
}

type Publisher struct {
	fs      afero.Afero
	mounter mount.Interface
	path    metadata.PathResolver
}

func (pub *Publisher) PublishVolume(ctx context.Context, volumeCfg *csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	if err := pub.mountOneAgent(volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, "failed to mount osagent volume: "+err.Error())
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (pub *Publisher) mountOneAgent(volumeCfg *csivolumes.VolumeConfig) error {
	hostDir := pub.path.OsAgentDir(volumeCfg.DynakubeName)
	_ = pub.fs.MkdirAll(hostDir, os.ModePerm)

	if err := pub.fs.MkdirAll(volumeCfg.TargetPath, os.ModePerm); err != nil {
		log.Info("failed to create directory for host mount", "directory", hostDir)

		return err
	}

	if err := pub.mounter.Mount(hostDir, volumeCfg.TargetPath, "", []string{"bind"}); err != nil {
		_ = pub.mounter.Unmount(hostDir)

		log.Info("failed to mount directory for host mount", "directory", hostDir)

		return err
	}

	return nil
}
