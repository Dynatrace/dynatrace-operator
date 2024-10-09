package troubleshoot

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/google/go-containerregistry/pkg/authn"
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

type ImagePullFunc func(image string) error

func verifyAllImagesAvailable(ctx context.Context, baseLog logd.Logger, keychain authn.Keychain, transport *http.Transport, dk *dynakube.DynaKube) error {
	log := baseLog.WithName("imagepull")

	imagePullFunc := CreateImagePullFunc(ctx, keychain, transport)

	if dk.NeedsOneAgent() {
		verifyImageIsAvailable(log, imagePullFunc, dk, componentOneAgent, false)
		verifyImageIsAvailable(log, imagePullFunc, dk, componentCodeModules, true)
	}

	if dk.ActiveGate().IsEnabled() {
		verifyImageIsAvailable(log, imagePullFunc, dk, componentActiveGate, false)
	}

	return nil
}

func verifyImageIsAvailable(log logd.Logger, pullImage ImagePullFunc, dk *dynakube.DynaKube, comp component, proxyWarning bool) {
	image, isCustomImage := comp.getImage(dk)
	if comp.SkipImageCheck(image) {
		logErrorf(log, "Unknown %s image", comp.String())

		return
	}

	componentName := comp.Name(isCustomImage)
	logNewCheckf(log, "Verifying that %s image %s can be pulled ...", componentName, image)

	if image == "" {
		logInfof(log, "No %s image configured", componentName)

		return
	}

	if dk.HasProxy() && proxyWarning {
		logWarningf(log, "Proxy setting in Dynakube is ignored for %s image due to technical limitations.", componentName)
	}

	if getEnvProxySettings() != nil {
		logWarningf(log, "Proxy settings in environment might interfere when pulling %s image in troubleshoot mode.", componentName)
	}

	err := pullImage(image)
	if err != nil {
		logErrorf(log, "Pulling %s image %s failed: %v", componentName, image, err)
	} else {
		logOkf(log, "%s image %s can be successfully pulled", componentName, image)
	}
}

func CreateImagePullFunc(ctx context.Context, keychain authn.Keychain, transport *http.Transport) ImagePullFunc {
	return func(image string) error {
		return tryImagePull(ctx, keychain, transport, image)
	}
}

func tryImagePull(ctx context.Context, keychain authn.Keychain, transport *http.Transport, image string) error {
	imageReference, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	_, err = remote.Get(
		imageReference,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(keychain),
		remote.WithTransport(transport),
		remote.WithPlatform(arch.ImagePlatform),
	)
	if err != nil {
		return err
	}

	return nil
}
