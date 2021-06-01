package csidriver

import (
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
)

const (
	volumeId   = "a-volume-id"
	podUid     = "a-pod-uid"
	namespace  = "a-namespace"
	targetPath = "a-target-path"
)

func TestCSIDriverServer_ParsePublishVolumeRequest(t *testing.T) {
	t.Run(`No volume capability`, func(t *testing.T) {
		volumeCfg, err := parsePublishVolumeRequest(&csi.NodePublishVolumeRequest{})

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = Volume capability missing in request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`No volume id`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = Volume ID missing in request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`No target path`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
			VolumeId:         volumeId,
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = Target path missing in request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Access type is of type block access`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Block{
					Block: &csi.VolumeCapability_BlockVolume{},
				},
			},
			VolumeId:   volumeId,
			TargetPath: targetPath,
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = cannot have block access type")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Access type is not of type mount access`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
			VolumeId:         volumeId,
			TargetPath:       targetPath,
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = expecting to have mount access type")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`No volume context`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   volumeId,
			TargetPath: targetPath,
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = Publish context missing in request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Pod namespace missing from requests volume context`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:      volumeId,
			TargetPath:    targetPath,
			VolumeContext: map[string]string{},
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = No namespace included with request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Pod uid missing from requests volume context`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   volumeId,
			TargetPath: targetPath,
			VolumeContext: map[string]string{
				podNamespaceContextKey: namespace,
			},
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = No Pod UID included with request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`request is parsed correctly`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   volumeId,
			TargetPath: targetPath,
			VolumeContext: map[string]string{
				podNamespaceContextKey: namespace,
				podUIDContextKey:       podUid,
			},
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.NoError(t, err)
		assert.NotNil(t, volumeCfg)
		assert.Equal(t, podUid, volumeCfg.podUID)
		assert.Equal(t, namespace, volumeCfg.namespace)
		assert.Equal(t, volumeId, volumeCfg.volumeId)
		assert.Equal(t, targetPath, volumeCfg.targetPath)
	})
}
