package troubleshoot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
)

const (
	pullSecretSuffix = "-pull-secret"
)

var (
	removeSchemaRegex      = regexp.MustCompile("^.*//(.*)$")
	removeApiEndpointRegex = regexp.MustCompile("^(.*)/[^/]*$")
	registryRegex          = regexp.MustCompile(`^(.*)/linux.*$`)
	imageRegex             = regexp.MustCompile(`^.*/(linux.*)$`)
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
	if err != nil {
		return err
	}

	if err = checkComponentImagePullable(troubleshootCtx.httpClient, "OneAgent", pullSecret, dynakubeOneAgentImage); err != nil {
		return err
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
	if err != nil {
		return err
	}

	if err = checkComponentImagePullable(troubleshootCtx.httpClient, "ActiveGate", pullSecret, dynakubeActiveGateImage); err != nil {
		return err
	}

	return nil
}

func checkComponentImagePullable(httpClient *http.Client, componentName string, pullSecret string, componentImage string) error {
	// split activegate image into registry and image name
	componentRegistry, componentImage, componentVersion, err := splitImageName(componentImage)
	if err != nil {
		return err
	}
	logInfof("using '%s' on '%s' with version '%s' as %s image", componentImage, componentRegistry, componentVersion, componentName)

	imageWorks := false

	// parse docker config
	var result Auths
	json.Unmarshal([]byte(pullSecret), &result)

	for registry, endpoint := range result.Auths {
		logInfof("checking images for registry '%s'", registry)

		apiToken := base64.StdEncoding.EncodeToString([]byte(endpoint.Username + ":" + endpoint.Password))

		if statusCode, err := connectToDockerRegistry(httpClient, "HEAD", "https://"+registry+"/v2/", "Basic", apiToken); err != nil {
			logErrorf("registry '%s' unreachable", registry)
			continue
		} else {
			if statusCode != 200 {
				logErrorf("registry '%s' unreachable (%d)", registry, statusCode)
				continue
			} else {
				logInfof("registry '%s' is accessible", registry)
			}
		}

		if statusCode, err := connectToDockerRegistry(httpClient, "HEAD", "https://"+registry+"/v2/"+componentImage+"/manifests/"+componentVersion, "Basic", apiToken); err != nil {
			logErrorf("registry '%s' unreachable", registry)
			continue
		} else {
			if statusCode != 200 {
				logErrorf("image '%s' with version '%s' not found on registry '%s'", componentImage, componentVersion, registry)
				continue
			} else {
				logInfof("image '%s' with version '%s' exists on registry '%s", componentImage, componentVersion, registry)
			}
		}

		imageWorks = true
	}

	if imageWorks {
		logOkf("%s image '%s' found", componentName, componentRegistry+"/"+componentImage)
	} else {
		return fmt.Errorf("%s image '%s' missing", componentName, componentRegistry+"/"+componentImage)
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

func getOneAgentImageEndpoint(troubleshootCtx *troubleshootContext) string {
	customImage := ""
	imageEndpoint := ""
	version := ""

	sr := removeSchemaRegex.FindStringSubmatch(troubleshootCtx.dynakube.Spec.APIURL)
	er := removeApiEndpointRegex.FindStringSubmatch(sr[1])
	imageEndpoint = er[1] + "/linux/oneagent"

	if troubleshootCtx.dynakube.ClassicFullStackMode() {
		customImage = troubleshootCtx.dynakube.Spec.OneAgent.ClassicFullStack.Image
		version = troubleshootCtx.dynakube.Spec.OneAgent.ClassicFullStack.Version
	} else if troubleshootCtx.dynakube.CloudNativeFullstackMode() {
		customImage = troubleshootCtx.dynakube.Spec.OneAgent.CloudNativeFullStack.Image
		version = troubleshootCtx.dynakube.Spec.OneAgent.CloudNativeFullStack.Version
	} else if troubleshootCtx.dynakube.HostMonitoringMode() {
		customImage = troubleshootCtx.dynakube.Spec.OneAgent.HostMonitoring.Image
		version = troubleshootCtx.dynakube.Spec.OneAgent.HostMonitoring.Version
	}

	if customImage != "" {
		imageEndpoint = customImage
	} else if version != "" {
		imageEndpoint = imageEndpoint + ":" + version
	}

	logInfof("OneAgent image endpoint '%s'", imageEndpoint)
	return imageEndpoint
}

func getActiveGateImageEndpoint(troubleshootCtx *troubleshootContext) string {
	imageEndpoint := ""

	sr := removeSchemaRegex.FindStringSubmatch(troubleshootCtx.dynakube.Spec.APIURL)
	er := removeApiEndpointRegex.FindStringSubmatch(sr[1])
	imageEndpoint = er[1] + "/linux/activegate"

	if troubleshootCtx.dynakube.Spec.ActiveGate.Image != "" {
		imageEndpoint = troubleshootCtx.dynakube.Spec.ActiveGate.Image
	}

	logInfof("ActiveGate image endpoint '%s'", imageEndpoint)
	return imageEndpoint
}

func splitImageName(imageName string) (registry string, image string, version string, err error) {
	err = nil

	registryMatches := registryRegex.FindStringSubmatch(imageName)
	if len(registryMatches) < 2 {
		err = fmt.Errorf("invalid image - registry not found (%s)", imageName)
		return
	}
	registry = registryRegex.FindStringSubmatch(imageName)[1]

	imageMatches := imageRegex.FindStringSubmatch(imageName)
	if len(imageMatches) < 2 {
		err = fmt.Errorf("invalid image - endpoint not found (%s)", imageName)
		return
	}
	image = imageRegex.FindStringSubmatch(imageName)[1]

	version = ""

	// check if image has version set
	fields := strings.Split(image, ":")
	if len(fields) == 1 || len(fields) >= 2 && fields[1] == "" {
		// no version set, default to latest
		version = "latest"
		logInfof("using latest image version")
	} else if len(fields) >= 2 {
		image = fields[0]
		version = fields[1]
		logInfof("using custom image version")
	} else {
		err = fmt.Errorf("invalid version of the image {\"image\": \"%s\"}", image)
	}
	return
}
