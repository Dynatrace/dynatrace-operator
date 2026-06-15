package csiserver

import (
	"errors"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes/app"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_Server_NodePublishVolume(t *testing.T) {
	request := func() *csi.NodePublishVolumeRequest {
		return &csi.NodePublishVolumeRequest{
			VolumeId:   "test-volume-id",
			TargetPath: "/test/target/path",
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
			VolumeContext: map[string]string{
				csivolumes.PodNameContextKey:               "test-pod",
				csivolumes.PodNamespaceContextKey:          "test-namespace",
				csivolumes.PodUIDContextKey:                "test-uid",
				csivolumes.CSIVolumeAttributeModeField:     appvolumes.Mode,
				csivolumes.CSIVolumeAttributeDynakubeField: "some-dynakube",
			},
		}
	}

	t.Run("no SetupWithManager => error", func(t *testing.T) {
		srv := NewServer(dtcsi.CSIOptions{})

		_, err := srv.NodePublishVolume(t.Context(), request())

		require.Error(t, err)
		require.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("authorizer denies => permission denied", func(t *testing.T) {
		srv := NewServer(dtcsi.CSIOptions{})
		auth := newMockVolumeAuthorizer(t)
		auth.EXPECT().Authorize(mock.Anything, mock.Anything).Return("", errors.New("denied"))
		srv.authorizer = auth

		_, err := srv.NodePublishVolume(t.Context(), request())

		require.Error(t, err)
		require.Equal(t, codes.PermissionDenied, status.Code(err))
	})
}
