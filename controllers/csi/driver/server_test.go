package csidriver

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCSIDriverServer_NodePublishVolume(t *testing.T) {
	server := &CSIDriverServer{}
	response, err := server.NodePublishVolume(context.TODO(), &csi.NodePublishVolumeRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, response)
}
