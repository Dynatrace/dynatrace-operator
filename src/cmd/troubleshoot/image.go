package troubleshoot

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
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
	log = newSubTestLogger("imagepull")

	if troubleshootCtx.dynakube.NeedsOneAgent() {
		verifyImageIsAvailable(troubleshootCtx, componentOneAgent, false)
		verifyImageIsAvailable(troubleshootCtx, componentCodeModules, true)
	}
	if troubleshootCtx.dynakube.NeedsActiveGate() {
		verifyImageIsAvailable(troubleshootCtx, componentActiveGate, false)
	}
	return nil
}

func verifyImageIsAvailable(troubleshootCtx *troubleshootContext, comp component, proxyWarning bool) {
	image, isCustomImage := comp.getImage(&troubleshootCtx.dynakube)
	if image == "" && comp != componentCodeModules {
		logErrorf("Unknown %s image", comp.String())
		return
	}

	componentName := comp.Name(isCustomImage)
	logNewCheckf("Verifying that %s image %s can be pulled ...", componentName, image)

	if image != "" {
		if troubleshootCtx.dynakube.HasProxy() && proxyWarning {
			logWarningf("Proxy setting in Dynakube is ignored for %s image due to technical limitations.", componentName)
		}

		if getEnvProxySettings() != nil {
			logWarningf("Proxy settings in environment might interfere when pulling %s image in troubleshoot mode.", componentName)
		}

		err := tryImagePull(troubleshootCtx, image)
		if err != nil {
			logErrorf("Pulling %s image %s failed: %v", componentName, image, err)
		} else {
			logOkf("%s image %s can be successfully pulled", componentName, image)
		}
	} else {
		logInfof("No %s image configured", componentName)
	}
}

func tryImagePull(troubleshootCtx *troubleshootContext, image string) error {
	imageReference, err := docker.ParseReference(normalizeDockerReference(image))
	if err != nil {
		return err
	}

	systemCtx, err := makeSysContext(troubleshootCtx, imageReference)
	if err != nil {
		return err
	}
	systemCtx.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue

	imageSource, err := imageReference.NewImageSource(troubleshootCtx.context, systemCtx)
	if err != nil {
		return err
	}
	defer imageSource.Close()

	return nil
}

func normalizeDockerReference(image string) string {
	return "//" + image
}

func makeSysContext(troubleshootCtx *troubleshootContext, imageReference types.ImageReference) (*types.SystemContext, error) {
	dockerCfg := dockerconfig.NewDockerConfig(troubleshootCtx.apiReader, troubleshootCtx.dynakube)
	err := dockerCfg.SetupAuthsFromSecret(&troubleshootCtx.pullSecret)
	if err != nil {
		return nil, err
	}
	return dockerconfig.MakeSystemContext(imageReference.DockerReference(), dockerCfg), nil
}

func getPullSecretToken(troubleshootCtx *troubleshootContext) (string, error) {
	secretBytes, hasPullSecret := troubleshootCtx.pullSecret.Data[dtpullsecret.DockerConfigJson]
	if !hasPullSecret {
		return "", fmt.Errorf("token .dockerconfigjson does not exist in secret '%s'", troubleshootCtx.pullSecret.Name)
	}

	secretStr := string(secretBytes)
	return secretStr, nil
}
