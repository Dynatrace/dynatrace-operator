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
	"path/filepath"
	"time"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mount "k8s.io/mount-utils"
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

func (publisher *AppVolumePublisher) PublishVolume(ctx context.Context, volumeCfg *csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	if publisher.hasRetryLimitReached(volumeCfg) {
		log.Info("reached max mount attempts for pod, attaching dummy volume, monitoring disabled", "pod", volumeCfg.PodName)

		return &csi.NodePublishVolumeResponse{}, nil
	}

	if !publisher.isArchiveAvailable(volumeCfg) {
		return nil, status.Error(
			codes.Unavailable,
			"version or digest is not yet set, csi-provisioner hasn't finished setup yet for DynaKube: "+volumeCfg.DynakubeName,
		)
	}

	if err := publisher.mountOneAgent(volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount oneagent volume: %s", err))
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// hasRetryLimitReached creates the base dir for a given app mount if it doesn't exist yet, checks the creation timestamp against the threshold
// if any of the FS calls fail in an unexpected way, then it is considered that the limit was reached.
func (publisher *AppVolumePublisher) hasRetryLimitReached(volumeCfg *csivolumes.VolumeConfig) bool {
	appDir := publisher.path.AppMountForID(volumeCfg.VolumeID)

	stat, err := publisher.fs.Stat(appDir)
	if errors.Is(err, os.ErrNotExist) {
		// First run, create folder, to keep track of time
		err := publisher.fs.MkdirAll(appDir, os.ModePerm)
		if err != nil {
			log.Error(err, "failed to create base dir for app mount, skipping injection", "dir", appDir)

			return true
		}

		return false
	} else if err != nil {
		log.Error(err, "unexpected failure in checking filesystem state, skipping injection", "dir", appDir)

		return true
	}

	limit := stat.ModTime().Add(volumeCfg.RetryTimeout)
	log.Info("not first attempt, time remaining before skipping injection", "time", time.Until(limit).String())

	return time.Now().After(limit) // TODO: User timeprovider for testing
}

// isArchiveAvailable checks if the LatestAgentBinaryForDynaKube folder exists or not
func (publisher *AppVolumePublisher) isArchiveAvailable(volumeCfg *csivolumes.VolumeConfig) bool {
	binDir := publisher.path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName)

	stat, err := publisher.fs.Stat(binDir)
	if errors.Is(err, os.ErrNotExist) {
		log.Info("no OneAgent binary is available to mount yet, will retry later", "dynakube", volumeCfg.DynakubeName)

		return false
	} else if err != nil {
		log.Error(err, "unexpected failure while checking for the latest OneAgent binary", "dynakube", volumeCfg.DynakubeName)

		return false
	}

	return stat.IsDir()
}

func (publisher *AppVolumePublisher) mountOneAgent(volumeCfg *csivolumes.VolumeConfig) error {
	mappedDir := publisher.path.AppMountMappedDir(volumeCfg.VolumeID)
	_ = publisher.fs.MkdirAll(mappedDir, os.ModePerm)

	upperDir, err := publisher.prepareUpperDir(volumeCfg)
	if err != nil {
		return err
	}

	workDir := publisher.path.AppMountWorkDir(volumeCfg.VolumeID)
	_ = publisher.fs.MkdirAll(workDir, os.ModePerm)

	lowerDir := publisher.path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName)

	linker, ok := publisher.fs.Fs.(afero.LinkReader)
	if ok { // will only be !ok during unit testing
		lowerDir, err = linker.ReadlinkIfPossible(lowerDir)
		if err != nil {
			log.Info("failed to read symlink for latest binary", "symlink", lowerDir)

			return err
		}
	}

	overlayOptions := []string{
		"lowerdir=" + lowerDir,
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

func (publisher *AppVolumePublisher) prepareUpperDir(volumeCfg *csivolumes.VolumeConfig) (string, error) {
	upperDir := publisher.path.AppMountVarDir(volumeCfg.VolumeID)
	err := publisher.fs.MkdirAll(upperDir, os.ModePerm)

	if err != nil {
		return "", errors.WithMessagef(err, "failed create overlay upper directory structure, path: %s", upperDir)
	}

	destAgentConfPath := publisher.path.OverlayVarRuxitAgentProcConf(volumeCfg.VolumeID)

	err = publisher.fs.MkdirAll(filepath.Dir(destAgentConfPath), os.ModePerm)
	if err != nil {
		return "", errors.WithMessagef(err, "failed create overlay upper directory agent config directory structure, path: %s", upperDir)
	}

	srcAgentConfPath := publisher.path.AgentSharedRuxitAgentProcConf(volumeCfg.DynakubeName)
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
