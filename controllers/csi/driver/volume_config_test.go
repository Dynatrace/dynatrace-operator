package csidriver

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
)

const (
	testId        = "test-id"
	testUid       = "test-uid"
	testNamespace = "test-namespace"
	testTarget    = "target"
	testFlavor    = "test-flavor"
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
			VolumeId:         testId,
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
			VolumeId:   testId,
			TargetPath: testTarget,
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = cannot have block access type")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Access type is not of type mount access`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
			VolumeId:         testId,
			TargetPath:       testTarget,
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
			VolumeId:   testId,
			TargetPath: testTarget,
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
			VolumeId:      testId,
			TargetPath:    testTarget,
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
			VolumeId:   testId,
			TargetPath: testTarget,
			VolumeContext: map[string]string{
				podNamespaceContextKey: testNamespace,
			},
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = No Pod UID included with request")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Pod flavor is neither default nor musl`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testId,
			TargetPath: testTarget,
			VolumeContext: map[string]string{
				podNamespaceContextKey: testNamespace,
				podUIDContextKey:       testUid,
				podFlavorContextKey:    testFlavor,
			},
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = invalid flavor in request: test-flavor")
		assert.Nil(t, volumeCfg)
	})
	t.Run(`Pod flavor can be either default or musl`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testId,
			TargetPath: testTarget,
			VolumeContext: map[string]string{
				podNamespaceContextKey: testNamespace,
				podUIDContextKey:       testUid,
				podFlavorContextKey:    dtclient.FlavorDefault,
			},
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.NoError(t, err)
		assert.NotNil(t, volumeCfg)
		assert.Equal(t, dtclient.FlavorDefault, volumeCfg.flavor)

		request.VolumeContext[podFlavorContextKey] = dtclient.FlavorMUSL
		volumeCfg, err = parsePublishVolumeRequest(request)

		assert.NoError(t, err)
		assert.NotNil(t, volumeCfg)
		assert.Equal(t, dtclient.FlavorMUSL, volumeCfg.flavor)

		delete(request.VolumeContext, podFlavorContextKey)
		volumeCfg, err = parsePublishVolumeRequest(request)

		assert.NoError(t, err)
		assert.NotNil(t, volumeCfg)
		assert.Equal(t, dtclient.FlavorDefault, volumeCfg.flavor)
	})
	t.Run(`request is parsed correctly`, func(t *testing.T) {
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testId,
			TargetPath: testTarget,
			VolumeContext: map[string]string{
				podNamespaceContextKey: testNamespace,
				podUIDContextKey:       testUid,
				podFlavorContextKey:    dtclient.FlavorMUSL,
			},
		}
		volumeCfg, err := parsePublishVolumeRequest(request)

		assert.NoError(t, err)
		assert.NotNil(t, volumeCfg)
		assert.Equal(t, dtclient.FlavorMUSL, volumeCfg.flavor)
		assert.Equal(t, testUid, volumeCfg.podUID)
		assert.Equal(t, testNamespace, volumeCfg.namespace)
		assert.Equal(t, testId, volumeCfg.volumeId)
		assert.Equal(t, testTarget, volumeCfg.targetPath)
	})
}
