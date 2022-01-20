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
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/container-storage-interface/spec/lib/go/csi"
	dto "github.com/prometheus/client_model/go"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewPublisher(client client.Client, fs afero.Afero, mounter mount.Interface, db metadata.Access, path metadata.PathResolver) csivolumes.Publisher {
	return &Publisher{
		client:  client,
		fs:      fs,
		mounter: mounter,
		db:      db,
		path:    path,
	}
}

type Publisher struct {
	client  client.Client
	fs      afero.Afero
	mounter mount.Interface
	db      metadata.Access
	path    metadata.PathResolver
}

func (pub *Publisher) PublishVolume(ctx context.Context, volumeCfg *csivolumes.VolumeConfig) (*csi.NodePublishVolumeResponse, error) {
	bindCfg, err := newBindConfig(ctx, pub, volumeCfg)
	if err != nil {
		return nil, err
	}

	if err := pub.mountOneAgent(bindCfg, volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount oneagent volume: %s", err))
	}

	if err := pub.storeVolume(bindCfg, volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to store volume info: %s", err))
	}
	agentsVersionsMetric.WithLabelValues(bindCfg.version).Inc()
	return &csi.NodePublishVolumeResponse{}, nil
}

func (pub *Publisher) UnpublishVolume(_ context.Context, volumeInfo *csivolumes.VolumeInfo) (*csi.NodeUnpublishVolumeResponse, error) {
	volume, err := pub.loadVolume(volumeInfo.VolumeId)
	if err != nil {
		log.Info("failed to load volume info", "error", err.Error())
	}

	overlayFSPath := pub.path.AgentRunDirForVolume(volume.TenantUUID, volumeInfo.VolumeId)

	if err = pub.umountOneAgent(volumeInfo.TargetPath, overlayFSPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount oneagent volume: %s", err.Error()))
	}

	if err = pub.db.DeleteVolume(volume.VolumeID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Info("deleted volume info", "ID", volume.VolumeID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)

	if err = pub.fs.RemoveAll(volumeInfo.TargetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info("volume has been unpublished", "targetPath", volumeInfo.TargetPath)

	pub.fireVolumeUnpublishedMetric(*volume)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (pub *Publisher) fireVolumeUnpublishedMetric(volume metadata.Volume) {
	if len(volume.Version) > 0 {
		agentsVersionsMetric.WithLabelValues(volume.Version).Dec()
		var m = &dto.Metric{}
		if err := agentsVersionsMetric.WithLabelValues(volume.Version).Write(m); err != nil {
			log.Error(err, "failed to get the value of agent version metric")
		}
		if m.Gauge.GetValue() <= float64(0) {
			agentsVersionsMetric.DeleteLabelValues(volume.Version)
		}
	}
}

func (pub *Publisher) mountOneAgent(bindCfg *bindConfig, volumeCfg *csivolumes.VolumeConfig) error {
	mappedDir := pub.path.OverlayMappedDir(bindCfg.tenantUUID, volumeCfg.VolumeId)
	_ = pub.fs.MkdirAll(mappedDir, os.ModePerm)

	upperDir := pub.path.OverlayVarDir(bindCfg.tenantUUID, volumeCfg.VolumeId)
	_ = pub.fs.MkdirAll(upperDir, os.ModePerm)

	workDir := pub.path.OverlayWorkDir(bindCfg.tenantUUID, volumeCfg.VolumeId)
	_ = pub.fs.MkdirAll(workDir, os.ModePerm)

	overlayOptions := []string{
		"lowerdir=" + pub.path.AgentBinaryDirForVersion(bindCfg.tenantUUID, bindCfg.version),
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

	return nil
}

func (pub *Publisher) umountOneAgent(targetPath string, overlayFSPath string) error {
	if err := pub.mounter.Unmount(targetPath); err != nil {
		log.Error(err, "Unmount failed", "path", targetPath)
	}

	if filepath.IsAbs(overlayFSPath) {
		agentDirectoryForPod := filepath.Join(overlayFSPath, dtcsi.OverlayMappedDirPath)
		if err := pub.mounter.Unmount(agentDirectoryForPod); err != nil {
			log.Error(err, "Unmount failed", "path", agentDirectoryForPod)
		}
	}

	return nil
}

func (pub *Publisher) storeVolume(bindCfg *bindConfig, volumeCfg *csivolumes.VolumeConfig) error {
	volume := metadata.NewVolume(volumeCfg.VolumeId, volumeCfg.PodName, bindCfg.version, bindCfg.tenantUUID)
	log.Info("inserting volume info", "ID", volume.VolumeID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)
	return pub.db.InsertVolume(volume)
}

func (pub *Publisher) loadVolume(volumeID string) (*metadata.Volume, error) {
	volume, err := pub.db.GetVolume(volumeID)
	if err != nil {
		return nil, err
	}
	if volume == nil {
		return &metadata.Volume{}, nil
	}
	log.Info("loaded volume info", "id", volume.VolumeID, "pod name", volume.PodName, "version", volume.Version, "dynakube", volume.TenantUUID)
	return volume, nil
}
