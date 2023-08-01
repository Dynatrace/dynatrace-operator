package image

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/installer/zip"
	"github.com/containers/image/v5/docker/reference"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type Properties struct {
	ImageUri     string
	DockerConfig dockerconfig.DockerConfig
	PathResolver metadata.PathResolver
	Metadata     metadata.Access
	ImageDigest  string
}

func GetDigest(uri string) (string, error) {
	ref, err := reference.Parse(uri)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("failed to parse image reference to create image installer, received imageUri: %s", uri))
	}
	canonRef, ok := ref.(reference.Canonical)
	if !ok {
		return "", errors.Errorf("unexpected type of image reference provided to image installer, expected reference with digest but received %s", uri)
	}
	return canonRef.Digest().Encoded(), nil
}

func NewImageInstaller(fs afero.Fs, props *Properties, transport *http.Transport) installer.Installer {
	if transport == nil {
		transport = &http.Transport{}
	}
	if props.DockerConfig.Dynakube.HasProxy() {
		proxyUrl, err := url.Parse(props.DockerConfig.Dynakube.Spec.Proxy.Value)
		if err != nil {
			log.Info("invalid proxy spec", "proxy", props.DockerConfig.Dynakube.Spec.Proxy.Value)
			return nil
		}
		log.Info("proxy spec", "proxy", props.DockerConfig.Dynakube.Spec.Proxy.Value, "proxyURL", proxyUrl.String(), "proxyURL.Host", proxyUrl.Host, "proxyURL.Port()", proxyUrl.Port())

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			proxyUrlName := ""
			if proxyUrl != nil {
				proxyUrlName = proxyUrl.String()
			}
			log.Info("via proxy", "proxyURL", proxyUrlName, "req.URL", req.URL.String(), "req.url.Scheme", req.URL.Scheme, "req.url.Host", req.URL.Host, "req.url.Port", req.URL.Port(), "req.User-Agent", req.Header.Get("User-Agent"))
			return proxyUrl, nil
		}
		transport.OnProxyConnectResponse = func(ctx context.Context, proxyURL *url.URL, connectReq *http.Request, connectRes *http.Response) error {
			log.Info("OnProxyConnectResponse", "proxyURL", proxyURL, "connectReq.URL", connectReq.URL.String(), "connectReq.User-Agent", connectReq.Header.Get("User-Agent"), "connectRes", connectRes.Status, "connectRes.Request.URL", connectRes.Request.URL.String())
			return nil
		}
	}

	if props.DockerConfig.Dynakube.Spec.TrustedCAs != "" {
		err := props.DockerConfig.StoreRequiredFiles(context.TODO(), afero.Afero{Fs: fs})
		if err != nil {
			return nil
		}

		trustedCAs, err := props.DockerConfig.Dynakube.TrustedCAs(context.TODO(), props.DockerConfig.ApiReader)
		if err != nil {
			return nil
		}

		rootCAs := x509.NewCertPool()
		if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
			log.Info("failed to append custom certs!")
		}
		transport.TLSClientConfig.RootCAs = rootCAs
	}

	return &Installer{
		fs:         fs,
		extractor:  zip.NewOneAgentExtractor(fs, props.PathResolver),
		props:      props,
		httpClient: &http.Client{Transport: transport},
	}
}

type Installer struct {
	fs         afero.Fs
	extractor  zip.Extractor
	props      *Properties
	httpClient *http.Client
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

func (installer Installer) Cleanup() error {
	return installer.props.DockerConfig.Cleanup(afero.Afero{Fs: installer.fs})
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
		&installer.props.DockerConfig,
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
