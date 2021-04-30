package csidriver

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	testId = "test-id"
	//testUid       = "test-uid"
	testNamespace = "test-namespace"
	testTarget    = "target"
)

func TestCSIDriverServer_NodePublishVolume(t *testing.T) {
	t.Run(`No volume capability`, func(t *testing.T) {
		server := &CSIDriverServer{}
		response, err := server.NodePublishVolume(context.TODO(), &csi.NodePublishVolumeRequest{})

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = Volume capability missing in request")
		assert.Nil(t, response)
	})
	t.Run(`No volume id`, func(t *testing.T) {
		server := &CSIDriverServer{}
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
		}
		response, err := server.NodePublishVolume(context.TODO(), request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = Volume ID missing in request")
		assert.Nil(t, response)
	})
	t.Run(`No target path`, func(t *testing.T) {
		server := &CSIDriverServer{}
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
			VolumeId:         testId,
		}
		response, err := server.NodePublishVolume(context.TODO(), request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = Target path missing in request")
		assert.Nil(t, response)
	})
	t.Run(`Access type is of type block access`, func(t *testing.T) {
		server := &CSIDriverServer{}
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Block{
					Block: &csi.VolumeCapability_BlockVolume{},
				},
			},
			VolumeId:   testId,
			TargetPath: testTarget,
		}
		response, err := server.NodePublishVolume(context.TODO(), request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = cannot have block access type")
		assert.Nil(t, response)
	})
	t.Run(`Access type is not of type mount access`, func(t *testing.T) {
		server := &CSIDriverServer{}
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{},
			VolumeId:         testId,
			TargetPath:       testTarget,
		}
		response, err := server.NodePublishVolume(context.TODO(), request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = expecting to have mount access type")
		assert.Nil(t, response)
	})
	t.Run(`No volume context`, func(t *testing.T) {
		server := &CSIDriverServer{}
		request := &csi.NodePublishVolumeRequest{
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeId:   testId,
			TargetPath: testTarget,
		}
		response, err := server.NodePublishVolume(context.TODO(), request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = Publish context missing in request")
		assert.Nil(t, response)
	})
	t.Run(`Pod namespace missing from requests volume context`, func(t *testing.T) {
		server := &CSIDriverServer{}
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
		response, err := server.NodePublishVolume(context.TODO(), request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = No namespace included with request")
		assert.Nil(t, response)
	})
	t.Run(`Pod uid missing from requests volume context`, func(t *testing.T) {
		server := &CSIDriverServer{}
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
		response, err := server.NodePublishVolume(context.TODO(), request)

		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = No Pod UID included with request")
		assert.Nil(t, response)
	})
	//t.Run(`Pod uid missing from requests volume context`, func(t *testing.T) {
	//	server := &CSIDriverServer{}
	//	request := &csi.NodePublishVolumeRequest{
	//		VolumeCapability: &csi.VolumeCapability{
	//			AccessType: &csi.VolumeCapability_Mount{
	//				Mount: &csi.VolumeCapability_MountVolume{},
	//			},
	//		},
	//		VolumeId: testId,
	//		TargetPath: testTarget,
	//		VolumeContext: map[string]string{
	//			podNamespaceContextKey: testNamespace,
	//			podUIDContextKey:       testUid,
	//		},
	//	}
	//	response, err := server.NodePublishVolume(context.TODO(), request)
	//
	//	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = No Pod UID included with request")
	//	assert.Nil(t, response)
	//})
}
