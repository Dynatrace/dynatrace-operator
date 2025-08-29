package image

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/zip"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testImageURL      = "test:5000/repo@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	testImageDigest   = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	emptyDockerConfig = "{\"auths\":{}}"
)

func TestIsAlreadyPresent(t *testing.T) {
	imageDigest := "test"
	pathResolver := metadata.PathResolver{}

	t.Run("returns early if path doesn't exist", func(t *testing.T) {
		installer := Installer{
			fs: afero.NewMemMapFs(),
			props: &Properties{
				PathResolver: pathResolver,
			},
		}
		isDownloaded := installer.isAlreadyPresent(pathResolver.AgentSharedBinaryDirForAgent(imageDigest))
		assert.False(t, isDownloaded)
	})
	t.Run("returns true if path present", func(t *testing.T) {
		installer := Installer{
			fs: testFileSystemWithSharedDirPresent(pathResolver, imageDigest),
			props: &Properties{
				PathResolver: pathResolver,
			},
		}
		isDownloaded := installer.isAlreadyPresent(pathResolver.AgentSharedBinaryDirForAgent(imageDigest))
		assert.True(t, isDownloaded)
	})
}

func testFileSystemWithSharedDirPresent(pathResolver metadata.PathResolver, imageDigest string) afero.Fs {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll(pathResolver.AgentSharedBinaryDirForAgent(imageDigest), 0777)

	return fs
}

func TestNewImageInstaller(t *testing.T) {
	ctx := context.Background()
	testFS := afero.NewMemMapFs()
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "dynakube",
		},
		Spec: dynakube.DynaKubeSpec{},
	}
	pullSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.PullSecretName(),
			Namespace: dk.Namespace,
		},
	}
	pullSecret.Data = map[string][]byte{
		corev1.DockerConfigJsonKey: []byte(emptyDockerConfig),
	}
	fakeClient := fake.NewClientWithIndex(&pullSecret)

	props := &Properties{
		PathResolver: metadata.PathResolver{RootDir: "/tmp"},
		ImageURI:     testImageURL,
		Dynakube:     dk,
		ImageDigest:  testImageDigest,
		APIReader:    fakeClient,
	}
	in, err := NewImageInstaller(ctx, testFS, props)
	require.NoError(t, err)
	assert.NotNil(t, in)
	assert.NotNil(t, in)
}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestInstaller_InstallAgent(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		fs        afero.Fs
		extractor zip.Extractor
		props     *Properties
		transport http.RoundTripper
	}

	type args struct {
		targetDir string
	}

	testFS := afero.NewMemMapFs()
	_, _ = afero.TempFile(testFS, "/dummy", "ioutil-test")
	transport := RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`OK`)),
		}
	})

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "Successfully install agent",
			fields: fields{
				fs:        testFS,
				extractor: nil,
				props: &Properties{
					PathResolver: metadata.PathResolver{RootDir: "/tmp"},
					ImageURI:     testImageURL,
					Dynakube: &dynakube.DynaKube{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "dynakube",
						},
						Spec: dynakube.DynaKubeSpec{},
					},
					ImageDigest: testImageDigest,
				},
				transport: transport,
			},
			args: args{targetDir: consts.AgentInitBinDirMount},
			want: true, wantErr: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := &Installer{
				fs:        tt.fields.fs,
				extractor: tt.fields.extractor,
				props:     tt.fields.props,
				transport: tt.fields.transport,
			}

			got, err := installer.InstallAgent(ctx, tt.args.targetDir)
			tt.wantErr(t, err, fmt.Sprintf("InstallAgent(%v)", tt.args.targetDir))
			assert.Equalf(t, tt.want, got, "InstallAgent(%v)", tt.args.targetDir)
		})
	}
}
