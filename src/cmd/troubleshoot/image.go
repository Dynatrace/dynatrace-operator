package troubleshoot

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/image"
	"github.com/pkg/errors"
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

func checkImagePullable(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("imagepull", true)

	if troubleshootCtx.dynakube.NeedsOneAgent() {
		err := checkOneAgentImagePullable(troubleshootCtx)
		if err != nil {
			return err
		}

		err = checkOneAgentCodeModulesImagePullable(troubleshootCtx)
		if err != nil {
			return err
		}
	}

	if troubleshootCtx.dynakube.NeedsActiveGate() {
		err := checkActiveGateImagePullable(troubleshootCtx)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkOneAgentImagePullable(troubleshootCtx *troubleshootContext) error {
	logNewTestf("checking if OneAgent image is pullable ...")

	pullSecret, err := getPullSecret(troubleshootCtx)
	if err != nil {
		return err
	}

	dynakubeOneAgentImage := getOneAgentImageEndpoint(troubleshootCtx)
	err = checkComponentImagePullable(troubleshootCtx.httpClient, "OneAgent", pullSecret, dynakubeOneAgentImage)

	if err != nil {
		return err
	}

	return nil
}

func checkOneAgentCodeModulesImagePullable(troubleshootCtx *troubleshootContext) error {
	logNewTestf("checking if OneAgent codeModules image is pullable ...")

	pullSecret, err := getPullSecret(troubleshootCtx)
	if err != nil {
		return err
	}

	dynakubeOneAgentCodeModulesImage := getOneAgentCodeModulesImageEndpoint(troubleshootCtx)

	if dynakubeOneAgentCodeModulesImage != "" {
		err = checkCustomModuleImagePullable(troubleshootCtx.httpClient, "OneAgentCodeModules", pullSecret, dynakubeOneAgentCodeModulesImage)

		if err != nil {
			return err
		}
	}
	return nil
}

func checkActiveGateImagePullable(troubleshootCtx *troubleshootContext) error {
	logNewTestf("checking if ActiveGate image is pullable ...")

	pullSecret, err := getPullSecret(troubleshootCtx)
	if err != nil {
		return err
	}

	dynakubeActiveGateImage := getActiveGateImageEndpoint(troubleshootCtx)
	err = checkComponentImagePullable(troubleshootCtx.httpClient, "ActiveGate", pullSecret, dynakubeActiveGateImage)

	if err != nil {
		return err
	}

	return nil
}

func checkComponentImagePullable(httpClient *http.Client, componentName string, pullSecret string, componentImage string) error {
	componentImageInfo, err := image.ComponentsFromUri(componentImage)

	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	logInfof("using '%s' on '%s' with version '%s' as %s image", componentImageInfo.Image, componentImageInfo.Registry, componentImageInfo.Version, componentName)

	var result Auths
	err = json.Unmarshal([]byte(pullSecret), &result)

	if err != nil {
		logErrorf("invalid pull secret, could not unmarshal to JSON: %v", err)
		return nil
	}

	for registry, credentials := range result.Auths {
		logInfof("checking images for registry '%s'", registry)

		err = registryAvailable(httpClient, registry+"/v2/", credentials.Auth)

		if err != nil {
			logErrorf("%v", err)
			continue
		}

		err = imageAvailable(httpClient, manifestUrl(registry, componentImageInfo), credentials.Auth)

		if err != nil {
			// Only print as a warning since other credentials might still work
			// At this point it is uncertain if there is an error with the credentials
			logWarningf("cannot pull image '%s' with version '%s' from registry '%s': %v",
				componentImageInfo.Image, componentImageInfo.Version, registry, err)
		} else {
			logOkf("image '%s' with version '%s' exists on registry '%s",
				componentImageInfo.Image, componentImageInfo.Version, registry)
			return nil
		}
	}

	// The image could not be pulled with any of the credentials
	// Return as an error
	return errors.New(fmt.Sprintf("%s image '%s' missing", componentName, componentImageInfo.Registry+"/"+componentImageInfo.Image))
}

func checkCustomModuleImagePullable(httpClient *http.Client, _ string, pullSecret string, codeModulesImage string) error {
	var result Auths
	err := json.Unmarshal([]byte(pullSecret), &result)

	if err != nil {
		return errors.Wrapf(err, "invalid pull secret, could not unmarshal to JSON")
	}

	codeModulesImageInfo, err := image.ComponentsFromUri(codeModulesImage)

	if err != nil {
		return err
	}

	logInfof("using '%s' on '%s' as OneAgentCodeModules image", codeModulesImage, codeModulesImageInfo.Registry)

	credentials, hasCredentials := result.Auths[codeModulesImageInfo.Registry]
	if !hasCredentials {
		credentials = Credentials{}
		// not returning an error because registry might be accessible without credentials
		logWarningf("no credentials for registry %s available", codeModulesImageInfo.Registry)
	}

	logInfof("checking images for registry '%s'", codeModulesImageInfo.Registry)

	err = registryAvailable(httpClient, codeModulesImageInfo.Registry, credentials.Auth)
	if err != nil {
		return err
	}

	logInfof("registry %s is accessible", codeModulesImageInfo.Registry)

	err = imageAvailable(httpClient, codemodulesImageUrl(codeModulesImageInfo), credentials.Auth)
	if err != nil {
		return errors.Wrapf(err, "image is missing, cannot pull image '%s' from registry '%s'", codeModulesImage, codeModulesImageInfo.Registry)
	}

	logOkf("OneAgentCodeModules image '%s' exists on registry '%s", codeModulesImageInfo.Image, codeModulesImageInfo.Registry)
	return nil
}

func imageAvailable(httpClient *http.Client, imageUrl string, apiToken string) error {
	statusCode, err := connectToDockerRegistry(httpClient, imageUrl, apiToken)

	if err != nil {
		return errors.Wrapf(err, "registry unreachable")
	} else if statusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("image not found (status code = %d)", statusCode))
	}

	return nil
}

func registryAvailable(httpClient *http.Client, registry string, apiToken string) error {
	statusCode, err := connectToDockerRegistry(httpClient, registryUrl(registry), apiToken)

	if err != nil {
		return errors.Wrapf(err, "registry '%s' unreachable", registry)
	} else if statusCode != http.StatusOK {
		// Don't fail immediately since connection works.
		// Maybe registry is not correctly implemented.
		logWarningf("registry '%s' is reachable but returned an unexpected error code (%d)", registry, statusCode)
	}

	return nil
}

func connectToDockerRegistry(httpClient *http.Client, httpUrl string, authToken string) (int, error) {
	req, err := http.NewRequest("HEAD", httpUrl, nil)

	if err != nil {
		return 0, err
	}

	if authToken != "" {
		req.Header.Set("Authorization", "Basic"+" "+authToken)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	return resp.StatusCode, nil
}

func getPullSecret(troubleshootCtx *troubleshootContext) (string, error) {
	secretBytes, hasPullSecret := troubleshootCtx.pullSecret.Data[dtpullsecret.DockerConfigJson]
	if !hasPullSecret {
		return "", errors.New(fmt.Sprintf("token .dockerconfigjson does not exist in secret '%s:%s'",
			troubleshootCtx.pullSecret.Namespace, troubleshootCtx.pullSecret.Name))
	}

	return string(secretBytes), nil
}

func manifestUrl(registry string, componentImageInfo image.Components) string {
	return fmt.Sprintf("%s/v2/%s/manifests/%s",
		registryUrl(registry), componentImageInfo.Image, componentImageInfo.Version)
}

func registryUrl(registry string) string {
	return fmt.Sprintf("https://%s", registry)
}

func codemodulesImageUrl(info image.Components) string {
	return fmt.Sprintf("https://%s/%s%s", info.Registry, info.Image, info.VersionUrlPostfix())
}
