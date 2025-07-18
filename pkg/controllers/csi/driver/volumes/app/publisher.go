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

package app

import (
	"context"
	"fmt"
	"os"
	"time"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mount "k8s.io/mount-utils"
)

func NewPublisher(fs afero.Afero, mounter mount.Interface, path metadata.PathResolver) csivolumes.Publisher {
	return &Publisher{
		fs:      fs,
		mounter: mounter,
		path:    path,
		time:    timeprovider.New(),
	}
}

type Publisher struct {
	fs      afero.Afero
	mounter mount.Interface
	time    *timeprovider.Provider
	path    metadata.PathResolver
}

func (pub *Publisher) PublishVolume(ctx context.Context, volumeCfg *csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	if pub.hasRetryLimitReached(volumeCfg) {
		log.Info("reached max mount attempts for pod, attaching dummy volume, monitoring disabled", "pod", volumeCfg.PodName)

		return &csi.NodePublishVolumeResponse{}, nil
	}

	if !pub.isCodeModuleAvailable(volumeCfg) {
		return nil, status.Error(
			codes.Unavailable,
			"version or digest is not yet set, csi-provisioner hasn't finished setup yet for DynaKube: "+volumeCfg.DynakubeName,
		)
	}

	if err := pub.mountCodeModule(volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount oneagent volume: %s", err))
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// hasRetryLimitReached creates the base dir for a given app mount if it doesn't exist yet, checks the creation timestamp against the threshold
// if any of the FS calls fail in an unexpected way, then it is considered that the limit was reached.
func (pub *Publisher) hasRetryLimitReached(volumeCfg *csivolumes.VolumeConfig) bool {
	appDir := pub.path.AppMountForID(volumeCfg.VolumeID)

	stat, err := pub.fs.Stat(appDir)
	if errors.Is(err, os.ErrNotExist) {
		// First run, create folder, to keep track of time
		err := pub.fs.MkdirAll(appDir, os.ModePerm)
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
	log.Info("not first attempt, time remaining before skipping injection", "remaining time", time.Until(limit).String())

	return pub.time.Now().After(limit)
}

// isCodeModuleAvailable checks if the LatestAgentBinaryForDynaKube folder exists or not
func (pub *Publisher) isCodeModuleAvailable(volumeCfg *csivolumes.VolumeConfig) bool {
	binDir := pub.path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName)

	stat, err := pub.fs.Stat(binDir)
	if errors.Is(err, os.ErrNotExist) {
		log.Info("no CodeModule is available to mount yet, will retry later", "dynakube", volumeCfg.DynakubeName)

		return false
	} else if err != nil {
		log.Error(err, "unexpected failure while checking for the latest available CodeModule", "dynakube", volumeCfg.DynakubeName)

		return false
	}

	return stat.IsDir()
}

func (pub *Publisher) mountCodeModule(volumeCfg *csivolumes.VolumeConfig) error {
	mappedDir := pub.path.AppMountMappedDir(volumeCfg.VolumeID)

	err := pub.fs.MkdirAll(mappedDir, os.ModePerm)
	if err != nil {
		return err
	}

	upperDir, err := pub.prepareUpperDir(volumeCfg)
	if err != nil {
		return err
	}

	workDir := pub.path.AppMountWorkDir(volumeCfg.VolumeID)

	err = pub.fs.MkdirAll(workDir, os.ModePerm)
	if err != nil {
		return err
	}

	lowerDir := pub.path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName)

	linker, ok := pub.fs.Fs.(afero.LinkReader)
	if ok { // will only be !ok during unit testing
		lowerDir, err = linker.ReadlinkIfPossible(lowerDir)
		if err != nil {
			log.Info("failed to read symlink for latest CodeModule", "symlink", lowerDir)

			return err
		}
	}

	overlayOptions := []string{
		"lowerdir=" + lowerDir,
		"upperdir=" + upperDir,
		"workdir=" + workDir,
	}

	if err := pub.fs.MkdirAll(volumeCfg.TargetPath, os.ModePerm); err != nil {
		return err
	}

	if err := pub.mounter.Mount("overlay", mappedDir, "overlay", overlayOptions); err != nil {
		return err
	}

	if err := pub.mounter.Mount(mappedDir, volumeCfg.TargetPath, "", []string{"bind"}); err != nil {
		_ = pub.mounter.Unmount(mappedDir)

		return err
	}

	err = pub.addPodInfoSymlink(volumeCfg)
	if err != nil {
		return err
	}

	return nil
}

func (pub *Publisher) addPodInfoSymlink(volumeCfg *csivolumes.VolumeConfig) error {
	appMountPodInfoDir := pub.path.AppMountPodInfoDir(volumeCfg.DynakubeName, volumeCfg.PodNamespace, volumeCfg.PodName)
	if err := pub.fs.MkdirAll(appMountPodInfoDir, os.ModePerm); err != nil {
		return err
	}

	targetDir := pub.path.AppMountForID(volumeCfg.VolumeID)

	if err := symlink.Remove(pub.fs.Fs, appMountPodInfoDir); err != nil {
		return err
	}

	err := symlink.Create(pub.fs.Fs, targetDir, appMountPodInfoDir)
	if err != nil {
		return err
	}

	return nil
}

func (pub *Publisher) prepareUpperDir(volumeCfg *csivolumes.VolumeConfig) (string, error) {
	upperDir := pub.path.AppMountVarDir(volumeCfg.VolumeID)

	err := pub.fs.MkdirAll(upperDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	err = pub.preparePodInfoUpperDir(volumeCfg)
	if err != nil {
		return "", err
	}

	return upperDir, nil
}

func (pub *Publisher) preparePodInfoUpperDir(volumeCfg *csivolumes.VolumeConfig) error {
	content := pub.path.AppMountPodInfoDir(volumeCfg.DynakubeName, volumeCfg.PodNamespace, volumeCfg.PodName)
	destPath := pub.path.OverlayVarPodInfo(volumeCfg.VolumeID)

	destFile, err := pub.fs.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return errors.WithMessagef(err, "failed to open destination pod-info file, path: %s", destPath)
	}

	_, err = destFile.WriteString(content)
	if err != nil {
		return errors.WithMessagef(err, "failed write into pod-info file, path: %s", destPath)
	}

	return nil
}
