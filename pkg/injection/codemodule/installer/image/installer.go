package image

import (
	"context"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/zip"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/registry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Properties struct {
	ImageURI     string
	APIReader    client.Reader
	Dynakube     *dynakube.DynaKube
	PathResolver metadata.PathResolver
	ImageDigest  string
}

func NewImageInstaller(ctx context.Context, props *Properties) (installer.Installer, error) {
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()

	transport, err := registry.PrepareTransportForDynaKube(ctx, props.APIReader, defaultTransport, props.Dynakube)
	if err != nil {
		return nil, err
	}

	keychain, err := dockerkeychain.NewDockerKeychains(ctx, props.APIReader, props.Dynakube.Namespace, props.Dynakube.PullSecretNames())
	if err != nil {
		return nil, err
	}

	return &Installer{
		extractor: zip.NewOneAgentExtractor(props.PathResolver),
		props:     props,
		transport: transport,
		keychain:  keychain,
	}, nil
}

type Installer struct {
	extractor zip.Extractor
	props     *Properties
	transport http.RoundTripper
	keychain  authn.Keychain
}

func (installer *Installer) InstallAgent(_ context.Context, targetDir string) (bool, error) {
	log.Info("installing agent from image")

	if installer.isAlreadyPresent(targetDir) {
		log.Info("agent already installed", "image", installer.props.ImageURI, "target dir", targetDir)

		return true, nil
	}

	err := os.MkdirAll(installer.props.PathResolver.AgentSharedBinaryDirBase(), common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create the base shared agent directory", "err", err)

		return false, errors.WithStack(err)
	}

	log.Info("installing agent", "image", installer.props.ImageURI, "target dir", targetDir)

	if err := installer.installAgentFromImage(targetDir); err != nil {
		_ = os.RemoveAll(targetDir)

		log.Info("failed to install agent from image", "err", err)

		return false, errors.WithStack(err)
	}

	if err := symlink.CreateForCurrentVersionIfNotExists(targetDir); err != nil {
		_ = os.RemoveAll(targetDir)

		log.Info("failed to create symlink for agent installation", "err", err)

		return false, errors.WithStack(err)
	}

	return true, nil
}

func (installer *Installer) installAgentFromImage(targetDir string) error {
	defer func() { _ = os.RemoveAll(CacheDir) }()

	err := os.MkdirAll(CacheDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create cache dir", "err", err)

		return errors.WithStack(err)
	}

	image := installer.props.ImageURI
	imageCacheDir := getCacheDirPath(installer.props.ImageDigest)

	err = installer.extractAgentBinariesFromImage(
		imagePullInfo{
			imageCacheDir: imageCacheDir,
			targetDir:     targetDir,
		},
		installer.props.ImageURI,
	)
	if err != nil {
		log.Info("failed to extract agent binaries from image via proxy", "image", image, "imageCacheDir", imageCacheDir, "err", err)
	}

	return nil
}

func (installer *Installer) isAlreadyPresent(targetDir string) bool {
	_, err := os.Stat(targetDir)

	return !os.IsNotExist(err)
}

func getCacheDirPath(digest string) string {
	return filepath.Join(CacheDir, digest)
}
