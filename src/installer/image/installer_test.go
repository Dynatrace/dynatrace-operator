package image

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/installer/zip"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
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
	fs.MkdirAll(pathResolver.AgentSharedBinaryDirForAgent(imageDigest), 0777)
	return fs
}

func TestGetDigest(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "basic digest from url",
			args:    args{uri: "test:5000/repo@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
			want:    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDigest(tt.args.uri)
			if !tt.wantErr(t, err, fmt.Sprintf("GetDigest(%v)", tt.args.uri)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetDigest(%v)", tt.args.uri)
		})
	}
}

func TestNewImageInstaller(t *testing.T) {
	testFS := afero.NewMemMapFs()
<<<<<<< HEAD
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "dynakube",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{},
	}
	pullSecret := dynakube.PullSecretWithoutData()
	pullSecret.Data = map[string][]byte{
		corev1.DockerConfigJsonKey: []byte(emptyDockerConfig),
	}
	fakeClient := fake.NewClientWithIndex(&pullSecret)
||||||| parent of 4c7e4959 (Update unit tests)
=======
	pullSecret := corev1.Secret{}
>>>>>>> 4c7e4959 (Update unit tests)

	props := &Properties{
		PathResolver: metadata.PathResolver{RootDir: "/tmp"},
		ImageUri:     testImageURL,
		Dynakube:     dynakube,
		ImageDigest:  testImageDigest,
		ApiReader:    fakeClient,
	}
<<<<<<< HEAD
	in, err := NewImageInstaller(testFS, props)
	require.NoError(t, err)
||||||| parent of 4c7e4959 (Update unit tests)
	in := NewImageInstaller(testFS, props)
=======
	in, err := NewImageInstaller(testFS, props, nil, pullSecret)
	require.NoError(t, err)
>>>>>>> 4c7e4959 (Update unit tests)
	assert.NotNil(t, in)
	assert.NotNil(t, in)
}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestInstaller_InstallAgent(t *testing.T) {
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
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Successfully install agent",
			fields: fields{
				fs:        testFS,
				extractor: nil,
				props: &Properties{
					PathResolver: metadata.PathResolver{RootDir: "/tmp"},
					ImageUri:     testImageURL,
					Dynakube: &dynatracev1beta1.DynaKube{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "dynakube",
						},
						Spec: dynatracev1beta1.DynaKubeSpec{},
					},
					ImageDigest: testImageDigest,
				},
				transport: transport,
			},
			args: args{targetDir: config.AgentBinDirMount},
			want: true, wantErr: assert.NoError,
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
			got, err := installer.InstallAgent(tt.args.targetDir)
			if !tt.wantErr(t, err, fmt.Sprintf("InstallAgent(%v)", tt.args.targetDir)) {
				return
			}
			assert.Equalf(t, tt.want, got, "InstallAgent(%v)", tt.args.targetDir)
		})
	}
}
