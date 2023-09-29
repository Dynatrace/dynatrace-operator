package troubleshoot

import (
	"context"
	"net/http"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	v1 "k8s.io/api/core/v1"
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

func verifyAllImagesAvailable(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, httpClient *http.Client, pullSecret v1.Secret, dynakube *dynatracev1beta1.DynaKube) error {
	log := baseLog.WithName("imagepull")

	if dynakube.NeedsOneAgent() {
		verifyImageIsAvailable(ctx, log, apiReader, httpClient, pullSecret, dynakube, componentOneAgent, false)
		verifyImageIsAvailable(ctx, log, apiReader, httpClient, pullSecret, dynakube, componentCodeModules, true)
	}
	if dynakube.NeedsActiveGate() {
		verifyImageIsAvailable(ctx, log, apiReader, httpClient, pullSecret, dynakube, componentActiveGate, false)
	}
	return nil
}

func verifyImageIsAvailable(ctx context.Context, log logr.Logger, apiReader client.Reader, httpClient *http.Client, pullSecret v1.Secret, dynakube *dynatracev1beta1.DynaKube, comp component, proxyWarning bool) {
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

	err := tryImagePull(ctx, apiReader, httpClient, pullSecret, dynakube, image)
	if err != nil {
		logErrorf(log, "Pulling %s image %s failed: %v", componentName, image, err)
	} else {
		logOkf(log, "%s image %s can be successfully pulled", componentName, image)
	}

}

func tryImagePull(ctx context.Context, apiReader client.Reader, httpClient *http.Client, pullSecret v1.Secret, dynakube *dynatracev1beta1.DynaKube, image string) error {
	imageReference, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	keychain, err := dockerkeychain.NewDockerKeychain(ctx, apiReader, pullSecret)
	if err != nil {
		return err
	}

	var transport *http.Transport
	if httpClient != nil && httpClient.Transport != nil {
		transport = httpClient.Transport.(*http.Transport).Clone()
	} else {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	transport, err = version.PrepareTransport(ctx, apiReader, transport, dynakube)
	if err != nil {
		return err
	}

	_, err = remote.Get(imageReference, remote.WithContext(ctx), remote.WithAuthFromKeychain(keychain), remote.WithTransport(transport))
	if err != nil {
		return err
	}
	return nil
}
