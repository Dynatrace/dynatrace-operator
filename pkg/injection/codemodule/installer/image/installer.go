package image

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/zip"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/dockerkeychain"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Properties struct {
	ImageUri     string
	ApiReader    client.Reader
	Dynakube     *dynatracev1beta1.DynaKube
	PathResolver metadata.PathResolver
	Metadata     metadata.Access
	ImageDigest  string
}

func GetDigest(uri string) (string, error) {
	ref, err := name.ParseReference(uri)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("failed to parse image reference to create image installer, received imageUri: %s", uri))
	}

	refDigest, ok := ref.(name.Digest)
	if !ok {
		return "", errors.Errorf("unexpected type of image reference provided to image installer, expected reference with digest but received %s", uri)
	}

	return strings.TrimLeft(refDigest.DigestStr(), digest.Canonical.String()+":"), nil
}

func NewImageInstaller(fs afero.Fs, props *Properties) (installer.Installer, error) {
	// Create default transport
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if props.Dynakube.HasProxy() {
		proxy, err := props.Dynakube.Proxy(context.TODO(), props.ApiReader)
		if err != nil {
			log.Info("failed to get proxy from dynakube", "proxy", proxy)
			return nil, err
		}

		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			log.Info("invalid proxy url", "proxy", proxy)
		}
		log.Info("proxy spec", "proxyURL.Host", proxyUrl.Host, "proxyURL.Port()", proxyUrl.Port())

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
	}

	if props.Dynakube.Spec.TrustedCAs != "" {
		trustedCAs, err := props.Dynakube.TrustedCAs(context.TODO(), props.ApiReader)
		if err != nil {
			return nil, err
		}

		rootCAs := x509.NewCertPool()
		if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
			log.Info("failed to append custom certs!")
		}

		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{} // nolint:gosec
		}
		transport.TLSClientConfig.RootCAs = rootCAs
	}

	keychain, err := dockerkeychain.NewDockerKeychain(context.TODO(), props.ApiReader, props.Dynakube.PullSecretWithoutData())
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

func (installer *Installer) InstallAgent(targetDir string) (bool, error) {
	log.Info("installing agent from image")

	if installer.isAlreadyPresent(targetDir) {
		log.Info("agent already installed", "target dir", targetDir)
		return false, nil
	}

	err := installer.fs.MkdirAll(installer.props.PathResolver.AgentSharedBinaryDirBase(), common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create the base shared agent directory", "err", err)
		return false, errors.WithStack(err)
	}

	log.Info("installing agent", "target dir", targetDir)
	if err := installer.installAgentFromImage(targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		log.Info("failed to install agent from image", "err", err)
		return false, errors.WithStack(err)
	}

	if err := symlink.CreateSymlinkForCurrentVersionIfNotExists(installer.fs, targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		log.Info("failed to create symlink for agent installation", "err", err)
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (installer *Installer) installAgentFromImage(targetDir string) error {
	defer installer.fs.RemoveAll(CacheDir)
	err := installer.fs.MkdirAll(CacheDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create cache dir", "err", err)
		return errors.WithStack(err)
	}
	image := installer.props.ImageUri

	if err != nil {
		log.Info("failed to get source information", "image", image)
		return errors.WithStack(err)
	}
	imageCacheDir := getCacheDirPath(installer.props.ImageDigest)
	if err != nil {
		log.Info("failed to get destination information", "image", image, "imageCacheDir", imageCacheDir)
		return errors.WithStack(err)
	}

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

func (installer Installer) isAlreadyPresent(targetDir string) bool {
	_, err := installer.fs.Stat(targetDir)
	return !os.IsNotExist(err)
}

func getCacheDirPath(digest string) string {
	return filepath.Join(CacheDir, digest)
}
