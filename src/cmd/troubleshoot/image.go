package troubleshoot

import (
	"context"
	"net/http"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/go-logr/logr"
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

func verifyAllImagesAvailable(ctx context.Context, baseLog logr.Logger, keychain authn.Keychain, transport *http.Transport, dynakube *dynatracev1beta1.DynaKube) error {
	log := baseLog.WithName("imagepull")

	imagePullFuncImpl := CreateImagePullFunc(ctx, keychain, transport)

	if dynakube.NeedsOneAgent() {
		verifyImageIsAvailable(log, imagePullFuncImpl, dynakube, componentOneAgent, false)
		verifyImageIsAvailable(log, imagePullFuncImpl, dynakube, componentCodeModules, true)
	}
	if dynakube.NeedsActiveGate() {
		verifyImageIsAvailable(log, imagePullFuncImpl, dynakube, componentActiveGate, false)
	}
	return nil
}

func verifyImageIsAvailable(log logr.Logger, pullImage imagePullFunc, dynakube *dynatracev1beta1.DynaKube, comp component, proxyWarning bool) {
	image, isCustomImage := comp.getImage(dynakube)
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

	if dynakube.HasProxy() && proxyWarning {
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

type imagePullFunc func(image string) error

func CreateImagePullFunc(ctx context.Context, keychain authn.Keychain, transport *http.Transport) imagePullFunc {
	return func(image string) error {
		return tryImagePull(ctx, keychain, transport, image)
	}
}

func tryImagePull(ctx context.Context, keychain authn.Keychain, transport *http.Transport, image string) error {
	imageReference, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	keychain, err := dockerkeychain.NewDockerKeychain(troubleshootCtx.context, troubleshootCtx.apiReader, troubleshootCtx.pullSecret)
	if err != nil {
		return err
	}

	transport, err = registry.PrepareTransport(ctx, apiReader, transport, dynakube)
	if err != nil {
		return err
	}

	_, err = remote.Get(imageReference, remote.WithContext(troubleshootCtx.context), remote.WithAuthFromKeychain(keychain), remote.WithTransport(transport))
	if err != nil {
		return err
	}
	return nil
}
