package troubleshoot

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pullSecretSuffix = "-pull-secret"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

type Endpoints map[string]Credentials

type Auths struct {
	Auths Endpoints `json:"auths"`
}

func verifyAllImagesAvailable(troubleshootCtx *troubleshootContext) error {
	log := troubleshootCtx.baseLog.WithName("imagepull")

	if troubleshootCtx.dynakube.NeedsOneAgent() {
		verifyImageIsAvailable(log, troubleshootCtx, componentOneAgent, false)
		verifyImageIsAvailable(log, troubleshootCtx, componentCodeModules, true)
	}
	if troubleshootCtx.dynakube.NeedsActiveGate() {
		verifyImageIsAvailable(log, troubleshootCtx, componentActiveGate, false)
	}
	return nil
}

func verifyImageIsAvailable(log logr.Logger, troubleshootCtx *troubleshootContext, comp component, proxyWarning bool) {
	image, isCustomImage := comp.getImage(&troubleshootCtx.dynakube)
	if comp.SkipImageCheck(image) {
		logErrorf(log, "Unknown %s image", comp.String())
		return
	}

	componentName := comp.Name(isCustomImage)
	logNewCheckf(log, "Verifying that %s image %s can be pulled ...", componentName, image)

	if image != "" {
		if troubleshootCtx.dynakube.HasProxy() && proxyWarning {
			logWarningf(log, "Proxy setting in Dynakube is ignored for %s image due to technical limitations.", componentName)
		}

		if getEnvProxySettings() != nil {
			logWarningf(log, "Proxy settings in environment might interfere when pulling %s image in troubleshoot mode.", componentName)
		}

		err := tryImagePull(troubleshootCtx, image)
		if err != nil {
			logErrorf(log, "Pulling %s image %s failed: %v", componentName, image, err)
		} else {
			logOkf(log, "%s image %s can be successfully pulled", componentName, image)
		}
	} else {
		logInfof(log, "No %s image configured", componentName)
	}
}

func tryImagePull(troubleshootCtx *troubleshootContext, image string) error {
	ref, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	dockerCfg := dockerconfig.NewDockerConfig(troubleshootCtx.apiReader, troubleshootCtx.dynakube)
	defer func(dockerCfg *dockerconfig.DockerConfig, fs afero.Afero) {
		_ = dockerCfg.Cleanup(fs)
	}(dockerCfg, troubleshootCtx.fs)

	dockerCfg.SetRegistryAuthSecret(&troubleshootCtx.pullSecret)

	err = dockerCfg.StoreRequiredFiles(troubleshootCtx.context, troubleshootCtx.fs)
	if err != nil {
		return err
	}

	keychain := dockerkeychain.NewDockerKeychain(dockerCfg.RegistryAuthPath, troubleshootCtx.fs)

	transport, err := createTransport(troubleshootCtx.dynakube, troubleshootCtx.context, troubleshootCtx.apiReader, troubleshootCtx.httpClient)
	if err != nil {
		return err
	}

	_, err = remote.Get(ref, remote.WithContext(troubleshootCtx.context), remote.WithAuthFromKeychain(keychain), remote.WithTransport(transport))
	if err != nil {
		return err
	}
	return nil
}

func createTransport(ctx context.Context, apiReader client.Reader, troubleShootHttpClient *http.Client, dynakube dynakubev1beta1.DynaKube) (*http.Transport, error) {
	var transport *http.Transport
	if troubleShootHttpClient != nil && troubleShootHttpClient.Transport != nil {
		transport = troubleShootHttpClient.Transport.(*http.Transport).Clone()
	} else {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	if kube.HasProxy() {
		proxy, err := kube.Proxy(ctx, apiReader)
		if err != nil {
			return nil, err
		}
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
	}

	if kube.Spec.TrustedCAs != "" {
		trustedCAs, err := kube.TrustedCAs(ctx, apiReader)
		if err != nil {
			return nil, err
		}
		transport, err = addCertificates(transport, trustedCAs)
		if err != nil {
			return nil, err
		}
	}
	return transport, nil
}

func addCertificates(transport *http.Transport, trustedCAs []byte) (*http.Transport, error) {
	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
		return nil, errors.New("failed to append custom certs!")
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{} //nolint:gosec
	}
	transport.TLSClientConfig.RootCAs = rootCAs

	return transport, nil
}

func getPullSecretToken(troubleshootCtx *troubleshootContext) (string, error) {
	secretBytes, hasPullSecret := troubleshootCtx.pullSecret.Data[dtpullsecret.DockerConfigJson]
	if !hasPullSecret {
		return "", fmt.Errorf("token .dockerconfigjson does not exist in secret '%s'", troubleshootCtx.pullSecret.Name)
	}

	secretStr := string(secretBytes)
	return secretStr, nil
}
