package csivolumes

import (
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testVolumeId   = "a-volume-id"
	testTargetPath = "a-target-path"
	testPodUID     = "a-pod-uid"
)

func TestCSIDriverServer_ParsePublishVolumeRequest(t *testing.T) {
	t.Run(`No volume capability`, func(t *testing.T) {
		volumeCfg, err := ParseNodePublishVolumeRequest(&csi.NodePublishVolumeRequest{})

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = Volume capability missing in request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`No volume id`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = Volume ID missing in request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`No target path`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
			VolumeId:         testVolumeId,
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = Target path missing in request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Access type is of type block access`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Block{
					Block: &csi.VolumeCapability_BlockVolume{},
				},
			},
			VolumeId:   testVolumeId,
			TargetPath: testTargetPath,
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = cannot have block access type")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Access type is not of type mount access`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
			VolumeId:         testVolumeId,
			TargetPath:       testTargetPath,
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = expecting to have mount access type")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`No volume context`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeId,
			TargetPath: testTargetPath,
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = Publish context missing in request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Pod name missing from requests volume context`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeId,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				CSIVolumeAttributeDynakubeField: testDynakubeName,
				CSIVolumeAttributeModeField:     "test",
				CSIVolumeAttributeRetryTimeout:  "5m",
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.Error(t, err)
		assert.Nil(t, volumeCfg)
	})
	t.Run(`mode missing from requests volume context`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeId,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				PodNameContextKey:               testPodUID,
				CSIVolumeAttributeDynakubeField: testDynakubeName,
				CSIVolumeAttributeRetryTimeout:  "5m",
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.Error(t, err)
		assert.Nil(t, volumeCfg)
	})
	t.Run(`dynakube missing from requests volume context`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeId,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				PodNameContextKey:              testPodUID,
				CSIVolumeAttributeModeField:    "test",
				CSIVolumeAttributeRetryTimeout: "5m",
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.Error(t, err)
		assert.Nil(t, volumeCfg)
	})
	t.Run(`retryTimeout missing from requests volume context`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeId,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				PodNameContextKey:               testPodUID,
				CSIVolumeAttributeDynakubeField: testDynakubeName,
				CSIVolumeAttributeModeField:     "test",
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.Error(t, err)
		assert.Nil(t, volumeCfg)
	})
	t.Run(`retryTimeout has incorrect format`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeId,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				PodNameContextKey:               testPodUID,
				CSIVolumeAttributeDynakubeField: testDynakubeName,
				CSIVolumeAttributeModeField:     "test",
				CSIVolumeAttributeRetryTimeout:  "5",
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.Error(t, err)
		assert.Nil(t, volumeCfg)
	})
	t.Run(`request is parsed correctly`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeId,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				PodNameContextKey:               testPodUID,
				CSIVolumeAttributeDynakubeField: testDynakubeName,
				CSIVolumeAttributeModeField:     "test",
				CSIVolumeAttributeRetryTimeout:  "5m",
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.NoError(t, err)
		assert.NotNil(t, volumeCfg)
		assert.Equal(t, testVolumeId, volumeCfg.VolumeID)
		assert.Equal(t, testTargetPath, volumeCfg.TargetPath)
		assert.Equal(t, testPodUID, volumeCfg.PodName)
		assert.Equal(t, "test", volumeCfg.Mode)
		assert.Equal(t, testDynakubeName, volumeCfg.DynakubeName)
		assert.Equal(t, time.Minute*5, volumeCfg.RetryTimeout)
	})
}
