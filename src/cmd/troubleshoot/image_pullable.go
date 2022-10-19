package troubleshoot

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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
	log = newTroubleshootLogger("[imagepull ] ")
	if err := addProxy(troubleshootCtx); err != nil {
		return err
	}

	if troubleshootCtx.dynakube.NeedsOneAgent() {
		verifyImageIsAvailable("OneAgent"+isCustomImage(troubleshootCtx.dynakube.CustomOneAgentImage()),
			troubleshootCtx.dynakube.ImmutableOneAgentImage(),
			troubleshootCtx)

		verifyImageIsAvailable("OneAgentCodeModules", troubleshootCtx.dynakube.CodeModulesImage(), troubleshootCtx)
	}

	if troubleshootCtx.dynakube.NeedsActiveGate() {
		verifyImageIsAvailable("ActiveGate"+isCustomImage(troubleshootCtx.dynakube.CustomActiveGateImage()),
			troubleshootCtx.dynakube.ActiveGateImage(),
			troubleshootCtx)
	}
	return nil
}

func isCustomImage(image string) string {
	if image != "" {
		return " (custom image)"
	}
	return ""
}

func verifyImageIsAvailable(component string, image string, troubleshootCtx *troubleshootContext) {
	logNewTestf("Verifying that %s image %s can be pulled ...", component, image)

	if image != "" {
		err := tryImagePull(troubleshootCtx, image)
		if err != nil {
			logErrorf("Pulling %s image %s failed: %v", component, image, err)
		} else {
			logOkf("%s image %s can be successfully pulled", component, image)
		}
	} else {
		logInfof("No %s image configured", component)
	}
}

func tryImagePull(troubleshootCtx *troubleshootContext, image string) error {
	imageReference, err := docker.ParseReference("//" + image)
	if err != nil {
		return err
	}

	systemCtx, err := makeSysContext(troubleshootCtx, imageReference)
	systemCtx.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	if err != nil {
		return err
	}

	imageSource, err := imageReference.NewImageSource(troubleshootCtx.ctx, systemCtx)
	if err != nil {
		return err
	}

	defer imageSource.Close()
	return nil
}

func makeSysContext(troubleshootCtx *troubleshootContext, imageReference types.ImageReference) (*types.SystemContext, error) {
	dockerCfg := dockerconfig.NewDockerConfig(troubleshootCtx.apiReader, troubleshootCtx.dynakube)
	err := dockerCfg.SetupAuthsFromSecret(&troubleshootCtx.pullSecret)
	if err != nil {
		return nil, err
	}
	return dockerconfig.MakeSystemContext(imageReference.DockerReference(), dockerCfg), nil
}

func addProxy(troubleshootCtx *troubleshootContext) error {
	proxyUrl := ""
	if troubleshootCtx.dynakube.Spec.Proxy != nil {
		if troubleshootCtx.dynakube.Spec.Proxy.Value != "" {
			proxyUrl = troubleshootCtx.dynakube.Spec.Proxy.Value
		} else if troubleshootCtx.dynakube.Spec.Proxy.ValueFrom != "" {
			var err error
			proxyUrl, err = kubeobjects.ExtractToken(&troubleshootCtx.proxySecret, dtclient.CustomProxySecretKey)
			if err != nil {
				return errorWithMessagef(err, "failed to extract proxy secret field")
			}
		}
	}
	if proxyUrl != "" {
		p, err := url.Parse(proxyUrl)
		if err != nil {
			return errorWithMessagef(err, "could not parse proxy URL!")
		}
		t := troubleshootCtx.httpClient.Transport.(*http.Transport)
		t.Proxy = http.ProxyURL(p)
		logInfof("using  '%s' proxy to connect to the registry", p.Host)
	}
	return nil
}

func getPullSecretToken(troubleshootCtx *troubleshootContext) (string, error) {
	secretBytes, ok := troubleshootCtx.pullSecret.Data[dtpullsecret.DockerConfigJson]
	if !ok {
		return "", fmt.Errorf("token .dockerconfigjson does not exist in secret '%s'", troubleshootCtx.pullSecretName)
	}

	secretStr := string(secretBytes)
	return secretStr, nil
}
