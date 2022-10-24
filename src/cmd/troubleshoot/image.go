package troubleshoot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
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
	log = newTroubleshootLogger("[imagepull ] ")

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
	componentImageInfo, err := splitImageName(componentImage)

	if err != nil {
		return err
	}

	logInfof("using '%s' on '%s' with version '%s' as %s image", componentImageInfo.image, componentImageInfo.registry, componentImageInfo.version, componentName)
	imageWorks := false

	// parse docker config
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
			logErrorf("cannot pull image '%s' with version '%s' from registry '%s': %v",
				componentImageInfo.image, componentImageInfo.version, registry, err)
		} else {
			logInfof("image '%s' with version '%s' exists on registry '%s",
				componentImageInfo.image, componentImageInfo.version, registry)
			imageWorks = true
		}
	}

	if imageWorks {
		logOkf("%s image '%s' found", componentName, componentImageInfo.registry+"/"+componentImageInfo.image)
	} else {
		return fmt.Errorf("%s image '%s' missing", componentName, componentImageInfo.registry+"/"+componentImageInfo.image)
	}

	return nil
}

func checkCustomModuleImagePullable(httpClient *http.Client, _ string, pullSecret string, codeModulesImage string) error {
	var result Auths
	err := json.Unmarshal([]byte(pullSecret), &result)

	if err != nil {
		return fmt.Errorf("invalid pull secret, could not unmarshal to JSON: %w", err)
	}

	codeModulesImageInfo, err := splitCustomImageName(codeModulesImage)
	if err != nil {
		return fmt.Errorf("invalid image URL: %w", err)
	}

	logInfof("using '%s' on '%s' as OneAgentCodeModules image", codeModulesImage, codeModulesImageInfo.registry)

	credentials, hasCredentials := result.Auths[codeModulesImageInfo.registry]
	if !hasCredentials {
		return fmt.Errorf("no credentials for registry %s available", codeModulesImageInfo.registry)
	}

	logInfof("checking images for registry '%s'", codeModulesImageInfo.registry)

	err = registryAvailable(httpClient, codeModulesImageInfo.registry, credentials.Auth)
	if err != nil {
		return err
	}

	logInfof("registry %s is accessible", codeModulesImageInfo.registry)

	err = imageAvailable(httpClient, codeModulesImageInfo.imageUrl(), credentials.Auth)
	if err != nil {
		return fmt.Errorf("image is missing, cannot pull image '%s' from registry '%s': %w", codeModulesImage, codeModulesImageInfo.registry, err)
	}

	logOkf("OneAgentCodeModules image '%s' exists on registry '%s", codeModulesImageInfo.image, codeModulesImageInfo.registry)
	return nil
}

func imageAvailable(httpClient *http.Client, imageUrl string, apiToken string) error {
	statusCode, err := connectToDockerRegistry(httpClient, "HEAD", imageUrl, "Basic", apiToken)

	if err != nil {
		return fmt.Errorf("registry unreachable: %w", err)
	} else if statusCode != http.StatusOK {
		return fmt.Errorf("image not found (status code = %d)", statusCode)
	}

	return nil
}

func registryAvailable(httpClient *http.Client, registry string, apiToken string) error {
	statusCode, err := connectToDockerRegistry(httpClient, "HEAD", registryUrl(registry), "Basic", apiToken)

	if err != nil {
		return fmt.Errorf("registry '%s' unreachable: %v", registry, err)
	} else if statusCode != http.StatusOK {
		return fmt.Errorf("registry '%s' unreachable (%d)", registry, statusCode)
	}

	return nil
}

func connectToDockerRegistry(httpClient *http.Client, httpMethod string, httpUrl string, authMethod string, authToken string) (int, error) {
	body := strings.NewReader("")
	req, err := http.NewRequest(httpMethod, httpUrl, body)

	if err != nil {
		return 0, err
	}

	if authMethod != "" && authToken != "" {
		req.Header.Set("Authorization", authMethod+" "+authToken)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer func() { _ = resp.Body.Close() }()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	return resp.StatusCode, nil
}

func getPullSecret(troubleshootCtx *troubleshootContext) (string, error) {
	secretBytes, hasPullSecret := troubleshootCtx.pullSecret.Data[dtpullsecret.DockerConfigJson]
	if !hasPullSecret {
		return "", fmt.Errorf("token .dockerconfigjson does not exist in secret '%s'", troubleshootCtx.pullSecretName)
	}

	return string(secretBytes), nil
}

func manifestUrl(registry string, componentImageInfo imageInfo) string {
	return fmt.Sprintf("%s/v2/%s/manifests/%s",
		registryUrl(registry), componentImageInfo.image, componentImageInfo.version)
}

func registryUrl(registry string) string {
	return fmt.Sprintf("https://%s", registry)
}
