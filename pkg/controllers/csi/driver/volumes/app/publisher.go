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
	"strings"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (publisher *AppVolumePublisher) PublishVolume(ctx context.Context, volumeCfg *csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	bindCfg, err := csivolumes.NewBindConfig(ctx, publisher.db, volumeCfg)
	if err != nil {
		return nil, err
	}

	hasTooManyAttempts, err := publisher.hasTooManyMountAttempts(ctx, bindCfg, volumeCfg)
	if err != nil {
		return nil, err
	}

	if hasTooManyAttempts {
		log.Info("reached max mount attempts for pod, attaching dummy volume, monitoring disabled", "pod", volumeCfg.PodName)

		return &csi.NodePublishVolumeResponse{}, nil
	}

	if !bindCfg.IsArchiveAvailable() {
		return nil, status.Error(
			codes.Unavailable,
			"version or digest is not yet set, csi-provisioner hasn't finished setup yet for tenant: "+bindCfg.TenantUUID,
		)
	}

	if err := publisher.ensureMountSteps(ctx, bindCfg, volumeCfg); err != nil {
		return nil, err
	}

	agentsVersionsMetric.WithLabelValues(bindCfg.MetricVersionLabel()).Inc()

	return &csi.NodePublishVolumeResponse{}, nil
}

func (publisher *AppVolumePublisher) UnpublishVolume(ctx context.Context, volumeInfo *csivolumes.VolumeInfo) (*csi.NodeUnpublishVolumeResponse, error) {
	volume, err := publisher.loadVolume(ctx, volumeInfo.VolumeID)
	if err != nil {
		log.Info("failed to load volume info", "error", err.Error())
	}

	if volume == nil {
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	log.Info("loaded volume info", "id", volume.VolumeID, "pod name", volume.PodName, "version", volume.Version, "dynakube", volume.TenantUUID)

	if volume.Version == "" {
		log.Info("requester has a dummy volume, no node-level unmount is needed")

		return &csi.NodeUnpublishVolumeResponse{}, publisher.db.DeleteVolume(ctx, volume.VolumeID)
	}

	overlayFSPath := publisher.path.AgentRunDirForVolume(volume.TenantUUID, volumeInfo.VolumeID)
	publisher.umountOneAgent(volumeInfo.TargetPath, overlayFSPath)

	if err = publisher.db.DeleteVolume(ctx, volume.VolumeID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info("deleted volume info", "ID", volume.VolumeID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)

	if err = publisher.fs.RemoveAll(volumeInfo.TargetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info("volume has been unpublished", "targetPath", volumeInfo.TargetPath)

	publisher.fireVolumeUnpublishedMetric(*volume)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (publisher *AppVolumePublisher) CanUnpublishVolume(ctx context.Context, volumeInfo *csivolumes.VolumeInfo) (bool, error) {
	volume, err := publisher.loadVolume(ctx, volumeInfo.VolumeID)
	if err != nil {
		return false, status.Error(codes.Internal, "failed to get volume info from database: "+err.Error())
	}

	return volume != nil, nil
}

func (publisher *AppVolumePublisher) fireVolumeUnpublishedMetric(volume metadata.Volume) {
	if len(volume.Version) > 0 {
		agentsVersionsMetric.WithLabelValues(volume.Version).Dec()

		var m = &dto.Metric{}
		if err := agentsVersionsMetric.WithLabelValues(volume.Version).Write(m); err != nil {
			log.Error(err, "failed to get the value of agent version metric")
		}

		if m.GetGauge().GetValue() <= float64(0) {
			agentsVersionsMetric.DeleteLabelValues(volume.Version)
		}
	}
}

func (publisher *AppVolumePublisher) buildLowerDir(bindCfg *csivolumes.BindConfig) string {
	var binFolderName string
	if bindCfg.ImageDigest == "" {
		binFolderName = bindCfg.Version
	} else {
		binFolderName = bindCfg.ImageDigest
	}

	directories := []string{
		publisher.path.AgentSharedBinaryDirForAgent(binFolderName),
	}

	return strings.Join(directories, ":")
}

func (publisher *AppVolumePublisher) prepareUpperDir(bindCfg *csivolumes.BindConfig, volumeCfg *csivolumes.VolumeConfig) (string, error) {
	upperDir := publisher.path.OverlayVarDir(bindCfg.TenantUUID, volumeCfg.VolumeID)
	err := publisher.fs.MkdirAll(upperDir, os.ModePerm)

	if err != nil {
		return "", errors.WithMessagef(err, "failed create overlay upper directory structure, path: %s", upperDir)
	}

	destAgentConfPath := publisher.path.OverlayVarRuxitAgentProcConf(bindCfg.TenantUUID, volumeCfg.VolumeID)

	err = publisher.fs.MkdirAll(filepath.Dir(destAgentConfPath), os.ModePerm)
	if err != nil {
		return "", errors.WithMessagef(err, "failed create overlay upper directory agent config directory structure, path: %s", upperDir)
	}

	srcAgentConfPath := publisher.path.AgentSharedRuxitAgentProcConf(bindCfg.TenantUUID, volumeCfg.DynakubeName)
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

func (publisher *AppVolumePublisher) mountOneAgent(bindCfg *csivolumes.BindConfig, volumeCfg *csivolumes.VolumeConfig) error {
	mappedDir := publisher.path.OverlayMappedDir(bindCfg.TenantUUID, volumeCfg.VolumeID)
	_ = publisher.fs.MkdirAll(mappedDir, os.ModePerm)

	upperDir, err := publisher.prepareUpperDir(bindCfg, volumeCfg)
	if err != nil {
		return err
	}

	workDir := publisher.path.OverlayWorkDir(bindCfg.TenantUUID, volumeCfg.VolumeID)
	_ = publisher.fs.MkdirAll(workDir, os.ModePerm)

	overlayOptions := []string{
		"lowerdir=" + publisher.buildLowerDir(bindCfg),
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

func (publisher *AppVolumePublisher) umountOneAgent(targetPath string, overlayFSPath string) {
	if err := publisher.mounter.Unmount(targetPath); err != nil {
		log.Error(err, "Unmount failed", "path", targetPath)
	}

	if filepath.IsAbs(overlayFSPath) {
		agentDirectoryForPod := filepath.Join(overlayFSPath, dtcsi.OverlayMappedDirPath)
		if err := publisher.mounter.Unmount(agentDirectoryForPod); err != nil {
			log.Error(err, "Unmount failed", "path", agentDirectoryForPod)
		}
	}
}

func (publisher *AppVolumePublisher) ensureMountSteps(ctx context.Context, bindCfg *csivolumes.BindConfig, volumeCfg *csivolumes.VolumeConfig) error {
	if err := publisher.mountOneAgent(bindCfg, volumeCfg); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to mount oneagent volume: %s", err))
	}

	if err := publisher.storeVolume(ctx, bindCfg, volumeCfg); err != nil {
		overlayFSPath := publisher.path.AgentRunDirForVolume(bindCfg.TenantUUID, volumeCfg.VolumeID)
		publisher.umountOneAgent(volumeCfg.TargetPath, overlayFSPath)

		return status.Error(codes.Internal, fmt.Sprintf("Failed to store volume info: %s", err))
	}

	return nil
}

func (publisher *AppVolumePublisher) hasTooManyMountAttempts(ctx context.Context, bindCfg *csivolumes.BindConfig, volumeCfg *csivolumes.VolumeConfig) (bool, error) {
	volume, err := publisher.loadVolume(ctx, volumeCfg.VolumeID)
	if err != nil {
		return false, err
	}

	if volume == nil {
		volume = createNewVolume(bindCfg, volumeCfg)
	}

	if volume.MountAttempts > bindCfg.MaxMountAttempts {
		return true, nil
	}

	volume.MountAttempts += 1

	return false, publisher.db.InsertVolume(ctx, volume)
}

func (publisher *AppVolumePublisher) storeVolume(ctx context.Context, bindCfg *csivolumes.BindConfig, volumeCfg *csivolumes.VolumeConfig) error {
	volume := createNewVolume(bindCfg, volumeCfg)
	log.Info("inserting volume info", "ID", volume.VolumeID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)

	return publisher.db.InsertVolume(ctx, volume)
}

func (publisher *AppVolumePublisher) loadVolume(ctx context.Context, volumeID string) (*metadata.Volume, error) {
	volume, err := publisher.db.GetVolume(ctx, volumeID)
	if err != nil {
		return nil, err
	}

	return volume, nil
}

func createNewVolume(bindCfg *csivolumes.BindConfig, volumeCfg *csivolumes.VolumeConfig) *metadata.Volume {
	version := bindCfg.Version
	if bindCfg.ImageDigest != "" {
		version = bindCfg.ImageDigest
	}

	return metadata.NewVolume(volumeCfg.VolumeID, volumeCfg.PodName, version, bindCfg.TenantUUID, 0)
}
