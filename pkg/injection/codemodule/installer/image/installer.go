package image

import (
	"context"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/zip"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Properties struct {
	ImageUri     string
	ApiReader    client.Reader
	Dynakube     *dynakube.DynaKube
	PathResolver metadata.PathResolver
	Metadata     metadata.Access
	ImageDigest  string
}

func NewImageInstaller(ctx context.Context, fs afero.Fs, props *Properties) (installer.Installer, error) {
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()

	transport, err := registry.PrepareTransportForDynaKube(ctx, props.ApiReader, defaultTransport, props.Dynakube)
	if err != nil {
		return nil, err
	}

	keychain, err := dockerkeychain.NewDockerKeychains(ctx, props.ApiReader, props.Dynakube.Namespace, props.Dynakube.PullSecretNames())
	if err != nil {
		return nil, err
	}

	return &Installer{
		fs:        fs,
		extractor: zip.NewOneAgentExtractor(fs, props.PathResolver),
		props:     props,
		transport: transport,
		keychain:  keychain,
	}, nil
}

type Installer struct {
	fs        afero.Fs
	extractor zip.Extractor
	props     *Properties
	transport http.RoundTripper
	keychain  authn.Keychain
}

func (installer *Installer) InstallAgent(_ context.Context, targetDir string) (bool, error) {
	log.Info("installing agent from image")

	if installer.isAlreadyPresent(targetDir) {
		log.Info("agent already installed", "image", installer.props.ImageUri, "target dir", targetDir)

		return false, nil
	}

	err := installer.fs.MkdirAll(installer.props.PathResolver.AgentSharedBinaryDirBase(), common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create the base shared agent directory", "err", err)

		return false, errors.WithStack(err)
	}

	log.Info("installing agent", "image", installer.props.ImageUri, "target dir", targetDir)

	if err := installer.installAgentFromImage(targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)

		log.Info("failed to install agent from image", "err", err)

		return false, errors.WithStack(err)
	}

	symlinkConfig := symlink.Config{
		ContextForLog:       "current version symlink",
		IsCurrentVerSymlink: true,
	}

	symlinkPath := filepath.Join(filepath.Join(targetDir, "/agent/bin"), "current")

	if err := symlink.Create(installer.fs, targetDir, symlinkPath, symlinkConfig); err != nil {
		_ = installer.fs.RemoveAll(targetDir)

		log.Info("failed to create symlink for agent installation", "err", err)

		return false, errors.WithStack(err)
	}

	return true, nil
}

func (installer *Installer) installAgentFromImage(targetDir string) error {
	defer func() { _ = installer.fs.RemoveAll(CacheDir) }()

	err := installer.fs.MkdirAll(CacheDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create cache dir", "err", err)

		return errors.WithStack(err)
	}

	image := installer.props.ImageUri
	imageCacheDir := getCacheDirPath(installer.props.ImageDigest)

	err = installer.extractAgentBinariesFromImage(
		imagePullInfo{
			imageCacheDir: imageCacheDir,
			targetDir:     targetDir,
		},
		installer.props.ImageUri,
	)
	if err != nil {
		log.Info("failed to extract agent binaries from image via proxy", "image", image, "imageCacheDir", imageCacheDir, "err", err)
	}

	return nil
}

func (installer *Installer) isAlreadyPresent(targetDir string) bool {
	_, err := installer.fs.Stat(targetDir)

	return !os.IsNotExist(err)
}

func getCacheDirPath(digest string) string {
	return filepath.Join(CacheDir, digest)
}
