package troubleshoot

import (
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
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
	imageReference, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	keychain, err := dockerkeychain.NewDockerKeychain(troubleshootCtx.context, troubleshootCtx.apiReader, troubleshootCtx.pullSecret)
	if err != nil {
		return err
	}

	var transport *http.Transport
	if troubleshootCtx.httpClient != nil && troubleshootCtx.httpClient.Transport != nil {
		transport = troubleshootCtx.httpClient.Transport.(*http.Transport).Clone()
	} else {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	transport, err = version.PrepareTransport(troubleshootCtx.context, troubleshootCtx.apiReader, transport, &troubleshootCtx.dynakube)
	if err != nil {
		return err
	}

	_, err = remote.Get(imageReference, remote.WithContext(troubleshootCtx.context), remote.WithAuthFromKeychain(keychain), remote.WithTransport(transport))
	if err != nil {
		return err
	}
	return nil
}
