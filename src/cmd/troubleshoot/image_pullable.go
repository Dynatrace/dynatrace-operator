package troubleshoot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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

	if err := addProxy(troubleshootCtx); err != nil {
		return err
	}

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

	pullSecret, err := getPullSecretToken(troubleshootCtx)
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

	pullSecret, err := getPullSecretToken(troubleshootCtx)
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

	pullSecret, err := getPullSecretToken(troubleshootCtx)
	if err != nil {
		return err
	}

	dynakubeActiveGateImage := getActiveGateImageEndpoint(troubleshootCtx)

	if err = checkComponentImagePullable(troubleshootCtx.httpClient, "ActiveGate", pullSecret, dynakubeActiveGateImage); err != nil {
		return err
	}
	return nil
}

func checkComponentImagePullable(httpClient *http.Client, componentName string, pullSecret string, componentImage string) error {
	// split image path into registry and image nam
	componentImageInfo, err := splitImageName(componentImage)
	//	componentRegistry, componentImage, componentVersion, err := splitImageName(componentImage)
	if err != nil {
		return err
	}
	logInfof("using '%s' on '%s' with version '%s' as %s image", componentImageInfo.image, componentImageInfo.registry, componentImageInfo.version, componentName)

	imageWorks := false

	// parse docker config
	var result Auths
	if err := json.Unmarshal([]byte(pullSecret), &result); err != nil {
		logErrorf("invalid pull secret, could not unmarshal to JSON: %v", err)
		return nil
	}

	for registry, credentials := range result.Auths {
		logInfof("checking images for registry '%s'", registry)

		apiToken := base64.StdEncoding.EncodeToString([]byte(credentials.Username + ":" + credentials.Password))

		if err := registryAvailable(httpClient, registry+"/v2/", apiToken); err != nil {
			logErrorf("%v", err)
			continue
		}

		if err := imageAvailable(httpClient, "https://"+registry+"/v2/"+componentImageInfo.image+"/manifests/"+componentImageInfo.version, apiToken); err != nil {
			logErrorf("cannot pull image '%s' with version '%s' from registry '%s': %v", componentImageInfo.image, componentImageInfo.version, registry, err)
			continue
		} else {
			logInfof("image '%s' with version '%s' exists on registry '%s", componentImageInfo.image, componentImageInfo.version, registry)
		}

		imageWorks = true
	}

	if imageWorks {
		logOkf("%s image '%s' found", componentName, componentImageInfo.registry+"/"+componentImageInfo.image)
	} else {
		return fmt.Errorf("%s image '%s' missing", componentName, componentImageInfo.registry+"/"+componentImageInfo.image)
	}
	return nil
}

func checkCustomModuleImagePullable(httpClient *http.Client, componentName string, pullSecret string, codeModulesImage string) error {
	// parse docker config
	var result Auths
	if err := json.Unmarshal([]byte(pullSecret), &result); err != nil {
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

	apiToken := base64.StdEncoding.EncodeToString([]byte(credentials.Username + ":" + credentials.Password))
	if err := registryAvailable(httpClient, codeModulesImageInfo.registry, apiToken); err != nil {
		return err
	}

	logInfof("registry %s is accessible", codeModulesImageInfo.registry)

	if err := imageAvailable(httpClient, "https://"+codeModulesImageInfo.registry+"/"+codeModulesImageInfo.image, apiToken); err != nil {
		return fmt.Errorf("image is missing, cannot pull image '%s' from registry '%s': %w", codeModulesImage, codeModulesImageInfo.registry, err)
	}

	logOkf("OneAgentCodeModules image '%s' exists on registry '%s", codeModulesImageInfo.image, codeModulesImageInfo.registry)
	return nil
}

func imageAvailable(httpClient *http.Client, imageUrl string, apiToken string) error {
	if statusCode, err := connectToDockerRegistry(httpClient, "HEAD", imageUrl, "Basic", apiToken); err != nil {
		return fmt.Errorf("registry unreachable: %w", err)
	} else {
		if statusCode != http.StatusOK {
			return fmt.Errorf("image not found (status code = %d)", statusCode)
		}
	}
	return nil
}

func registryAvailable(httpClient *http.Client, registry string, apiToken string) error {
	if statusCode, err := connectToDockerRegistry(httpClient, "HEAD", "https://"+registry, "Basic", apiToken); err != nil {
		return fmt.Errorf("registry '%s' unreachable: %v", registry, err)
	} else {
		if statusCode != http.StatusOK {
			return fmt.Errorf("registry '%s' unreachable (%d)", registry, statusCode)
		}
	}
	return nil
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
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}

func getPullSecretToken(troubleshootCtx *troubleshootContext) (string, error) {
	secretBytes, ok := troubleshootCtx.pullSecret.Data[dtpullsecret.DockerConfigJson]
	if !ok {
		return "", fmt.Errorf("token .dockerconfigjson does not exist in secret '%s'", troubleshootCtx.pullSecretName)
	}

	secretStr := string(secretBytes)
	return secretStr, nil
}
