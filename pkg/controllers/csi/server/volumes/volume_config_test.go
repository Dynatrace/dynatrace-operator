package csivolumes

import (
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testVolumeID     = "a-volume-id"
	testTargetPath   = "a-target-path"
	testPodUID       = "a-pod-uid"
	testNs           = "a-namespace"
	testDynakubeName = "a-dynakube"
)

func TestCSIDriverServer_ParsePublishVolumeRequest(t *testing.T) {
	t.Run("No volume id", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = Volume ID missing in request")
		assert.NotNil(t, volumeCfg)
	})
	t.Run("No target path", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeId: testVolumeID,
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = Target path missing in request")
		require.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
	})
	t.Run("No volume capability", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeId:   testVolumeID,
			TargetPath: testTargetPath,
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = Volume capability missing in request")
		require.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
		assert.Equal(t, request.GetTargetPath(), volumeCfg.TargetPath)
	})
	t.Run("Access type is of type block access", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Block{
					Block: &csi.VolumeCapability_BlockVolume{},
				},
			},
			VolumeId:   testVolumeID,
			TargetPath: testTargetPath,
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = cannot have block access type")
		require.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
		assert.Equal(t, request.GetTargetPath(), volumeCfg.TargetPath)
	})
	t.Run("Access type is not of type mount access", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
			VolumeId:         testVolumeID,
			TargetPath:       testTargetPath,
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = expecting to have mount access type")
		require.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
		assert.Equal(t, request.GetTargetPath(), volumeCfg.TargetPath)
	})
	t.Run("No volume context", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeID,
			TargetPath: testTargetPath,
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = Publish context missing in request")
		require.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
		assert.Equal(t, request.GetTargetPath(), volumeCfg.TargetPath)
	})
	t.Run("Pod name missing from requests volume context", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:      testVolumeID,
			TargetPath:    testTargetPath,
			VolumeContext: map[string]string{},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.Error(t, err)
		require.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
		assert.Equal(t, request.GetTargetPath(), volumeCfg.TargetPath)
	})
	t.Run("Pod namespace missing from requests volume context", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeID,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				PodNameContextKey: testPodUID,
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.Error(t, err)
		require.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
		assert.Equal(t, request.GetTargetPath(), volumeCfg.TargetPath)
		assert.Equal(t, request.GetVolumeContext()[PodNameContextKey], volumeCfg.PodName)
	})
	t.Run("mode missing from requests volume context", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeID,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				PodNameContextKey:      testPodUID,
				PodNamespaceContextKey: testNs,
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.Error(t, err)
		require.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
		assert.Equal(t, request.GetTargetPath(), volumeCfg.TargetPath)
		assert.Equal(t, request.GetVolumeContext()[PodNameContextKey], volumeCfg.PodName)
		assert.Equal(t, request.GetVolumeContext()[PodNamespaceContextKey], volumeCfg.PodNamespace)
	})
	t.Run("dk name missing from requests volume context", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeID,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				PodNameContextKey:           testPodUID,
				PodNamespaceContextKey:      testNs,
				CSIVolumeAttributeModeField: "test",
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.Error(t, err)
		require.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
		assert.Equal(t, request.GetTargetPath(), volumeCfg.TargetPath)
		assert.Equal(t, request.GetVolumeContext()[PodNameContextKey], volumeCfg.PodName)
		assert.Equal(t, request.GetVolumeContext()[PodNamespaceContextKey], volumeCfg.PodNamespace)
		assert.Equal(t, request.GetVolumeContext()[CSIVolumeAttributeModeField], volumeCfg.Mode)
	})

	t.Run("happy path", func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testVolumeID,
			TargetPath: testTargetPath,
			VolumeContext: map[string]string{
				PodNameContextKey:               testPodUID,
				PodNamespaceContextKey:          testNs,
				CSIVolumeAttributeModeField:     "test",
				CSIVolumeAttributeDynakubeField: "dk",
			},
		}
		volumeCfg, err := ParseNodePublishVolumeRequest(request)

		require.NoError(t, err)
		assert.NotNil(t, volumeCfg)
		assert.Equal(t, request.GetVolumeId(), volumeCfg.VolumeID)
		assert.Equal(t, request.GetTargetPath(), volumeCfg.TargetPath)
		assert.Equal(t, request.GetVolumeContext()[PodNameContextKey], volumeCfg.PodName)
		assert.Equal(t, request.GetVolumeContext()[PodNamespaceContextKey], volumeCfg.PodNamespace)
		assert.Equal(t, request.GetVolumeContext()[CSIVolumeAttributeModeField], volumeCfg.Mode)
		assert.Equal(t, request.GetVolumeContext()[CSIVolumeAttributeDynakubeField], volumeCfg.DynakubeName)
		assert.NotNil(t, volumeCfg.RetryTimeout)
	})
}
