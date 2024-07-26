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
	"os"
	"strings"
	"time"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
)

func NewAppVolumePublisher(fs afero.Afero, mounter mount.Interface, path metadata.PathResolver) csivolumes.Publisher {
	return &AppVolumePublisher{
		fs:      fs,
		mounter: mounter,
		path:    path,
	}
}

type AppVolumePublisher struct {
	fs      afero.Afero
	mounter mount.Interface
	path    metadata.PathResolver
}

func (publisher *AppVolumePublisher) PublishVolume(_ context.Context, volumeCfg csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	hasTooManyAttempts, err := publisher.hasTooManyMountAttempts(volumeCfg)
	if err != nil {
		return nil, err
	}

	if hasTooManyAttempts {
		log.Info("reached max mount attempts for pod, attaching dummy volume, monitoring disabled", "pod", volumeCfg.Pod)

		return &csi.NodePublishVolumeResponse{}, nil
	}

	if !publisher.IsArchiveAvailable(volumeCfg.Version) {
		return nil, status.Error(
			codes.Unavailable,
			"version or digest is not yet set, csi-provisioner hasn't finished setup yet",
		)
	}

	if err := publisher.mountOneAgent(volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount oneagent volume: %s", err))
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (publisher *AppVolumePublisher) IsArchiveAvailable(version string) bool {
	expectedPath := publisher.path.SharedCodeModulesDirForVersion(version)

	exists, err := publisher.fs.Exists(expectedPath)
	if err != nil {
		log.Info("failed to verify if codemodule at path exists", "path", expectedPath, "err", err.Error())
	}

	return exists
}

func (publisher *AppVolumePublisher) UnpublishVolume(_ context.Context, volumeInfo csivolumes.VolumeInfo) (*csi.NodeUnpublishVolumeResponse, error) {
	publisher.unmountOneAgent(volumeInfo.TargetPath)

	if err := publisher.fs.RemoveAll(volumeInfo.TargetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info("volume has been unpublished", "targetPath", volumeInfo.TargetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (publisher *AppVolumePublisher) CanUnpublishVolume(_ context.Context, volumeInfo csivolumes.VolumeInfo) (bool, error) { // probably not even needed
	return strings.Contains(volumeInfo.TargetPath, "oneagent-bin"), nil // TODO: fix to use constant, fix import cycle
}

func (publisher *AppVolumePublisher) prepareUpperDir(volumeCfg csivolumes.VolumeConfig) (string, error) {
	upperDir := publisher.path.OverlayVarDir(volumeCfg.Namespace, volumeCfg.Namespace, volumeCfg.VolumeID)
	err := publisher.fs.MkdirAll(upperDir, os.ModePerm)

	if err != nil {
		return "", errors.WithMessagef(err, "failed create overlay upper directory structure, path: %s", upperDir)
	}

	// destAgentConfPath := publisher.path.OverlayVarRuxitAgentProcConf(tenantConfig.TenantUUID, volumeCfg.VolumeID)

	// err = publisher.fs.MkdirAll(filepath.Dir(destAgentConfPath), os.ModePerm)
	// if err != nil {
	// 	return "", errors.WithMessagef(err, "failed create overlay upper directory agent config directory structure, path: %s", upperDir)
	// }

	// srcAgentConfPath := path.Join(tenantConfig.ConfigDirPath, processmoduleconfig.RuxitAgentProcPath)
	// srcFile, err := publisher.fs.Open(srcAgentConfPath)

	// if err != nil {
	// 	return "", errors.WithMessagef(err, "failed to open ruxitagentproc.conf file, path: %s", srcAgentConfPath)
	// }

	// defer func() { _ = srcFile.Close() }()

	// srcStat, err := srcFile.Stat()
	// if err != nil {
	// 	return "", errors.WithMessage(err, "failed to get source ruxitagentproc.conf file info")
	// }

	// destFile, err := publisher.fs.OpenFile(destAgentConfPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcStat.Mode())
	// if err != nil {
	// 	return "", errors.WithMessagef(err, "failed to open destination ruxitagentproc.conf file, path: %s", destAgentConfPath)
	// }

	// defer func() { _ = destFile.Close() }()

	// _, err = io.Copy(destFile, srcFile)
	// if err != nil {
	// 	return "", errors.WithMessagef(err, "failed to copy ruxitagentproc.conf file to overlay, from->to: %s -> %s", srcAgentConfPath, destAgentConfPath)
	// }

	return upperDir, nil
}

func (publisher *AppVolumePublisher) mountOneAgent(volumeCfg csivolumes.VolumeConfig) error {
	mappedDir := publisher.path.OverlayMappedDir(volumeCfg.Namespace, volumeCfg.Namespace, volumeCfg.VolumeID)
	_ = publisher.fs.MkdirAll(mappedDir, os.ModePerm)

	upperDir, err := publisher.prepareUpperDir(volumeCfg)
	if err != nil {
		return err
	}

	workDir := publisher.path.OverlayWorkDir(volumeCfg.Namespace, volumeCfg.Namespace, volumeCfg.VolumeID)
	_ = publisher.fs.MkdirAll(workDir, os.ModePerm)

	overlayOptions := []string{
		"lowerdir=" + publisher.path.SharedCodeModulesDirForVersion(volumeCfg.Version),
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

func (publisher *AppVolumePublisher) unmountOneAgent(targetPath string) {
	if err := publisher.mounter.Unmount(targetPath); err != nil {
		log.Error(err, "Unmount failed", "path", targetPath)
	}
	// if filepath.IsAbs(overlayFSPath) {
	// 	if err := publisher.mounter.Unmount(overlayFSPath); err != nil {
	// 		log.Error(err, "Unmount failed", "path", overlayFSPath)
	// 	}
	// }
}

func (publisher *AppVolumePublisher) hasTooManyMountAttempts(volumeCfg csivolumes.VolumeConfig) (bool, error) {
	baseDir := publisher.path.AppMountsDir(volumeCfg.Namespace, volumeCfg.Pod, volumeCfg.VolumeID)

	baseExists, err := publisher.fs.DirExists(baseDir)
	if err != nil {
		return true, err
	}

	if !baseExists {
		err := publisher.fs.MkdirAll(baseDir, os.ModePerm)
		if err != nil {
			return true, err
		}

		return false, nil // attempt 1.
	}

	stat, err := publisher.fs.Stat(baseDir)
	if err != nil {
		return true, err
	}

	return time.Now().After(stat.ModTime().Add(volumeCfg.Timeout)), nil // TODO: use timeprovider to test
}
