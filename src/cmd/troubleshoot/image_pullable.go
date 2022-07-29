package troubleshoot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	query := kubeobjects.NewDynakubeQuery(nil, troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(context.TODO())
	dynakube, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakubeName})
	if err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}

	if err := addProxy(troubleshootCtx); err != nil {
		return err
	}

	if dynakube.NeedsOneAgent() {
		err := checkOneAgentImagePullable(troubleshootCtx)
		if err != nil {
			return err
		}
	}

	if dynakube.NeedsActiveGate() {
		err := checkActiveGateImagePullable(troubleshootCtx)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkOneAgentImagePullable(troubleshootCtx *troubleshootContext) error {
	logNewTestf("checking if OneAgent image is pullable ...")

	pullSecretName, err := getPullSecretName(troubleshootCtx.apiReader, troubleshootCtx.dynakubeName, troubleshootCtx.namespaceName)
	if err != nil {
		return err
	}

	pullSecret, err := getPullSecret(troubleshootCtx.apiReader, pullSecretName, troubleshootCtx.namespaceName)
	if err != nil {
		return err
	}

	dynakubeOneAgentImage, err := getOneAgentImageEndpoint(troubleshootCtx)
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

	pullSecretName, err := getPullSecretName(troubleshootCtx.apiReader, troubleshootCtx.dynakubeName, troubleshootCtx.namespaceName)
	if err != nil {
		return err
	}
	pullSecret, err := getPullSecret(troubleshootCtx.apiReader, pullSecretName, troubleshootCtx.namespaceName)
	if err != nil {
		return err
	}

	dynakubeActiveGateImage, err := getActiveGateImageEndpoint(troubleshootCtx)
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
		logErrorf("%s image '%s' missing", componentName, componentRegistry+"/"+componentImage)
		return fmt.Errorf("%s image '%s' missing", componentName, componentRegistry+"/"+componentImage)
	}
	return nil
}

func addProxy(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewDynakubeQuery(nil, troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(context.TODO())
	dynakube, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakubeName})
	if err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}

	proxyUrl := ""
	if dynakube.Spec.Proxy != nil {
		if dynakube.Spec.Proxy.Value != "" {
			proxyUrl = dynakube.Spec.Proxy.Value
		} else if dynakube.Spec.Proxy.ValueFrom != "" {
			proxySecret := corev1.Secret{}
			if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: dynakube.Spec.Proxy.ValueFrom, Namespace: troubleshootCtx.namespaceName}, &proxySecret); err != nil {
				logWithErrorf(err, "'%s:%s' proxy secret is missing", dynakube.Spec.Proxy.ValueFrom, troubleshootCtx.dynatraceApiSecretName)
				return err
			}
			var err error
			proxyUrl, err = kubeobjects.ExtractToken(&proxySecret, dtclient.CustomProxySecretKey)
			if err != nil {
				logWithErrorf(err, "failed to extract proxy secret field")
				return fmt.Errorf("failed to extract proxy secret field")
			}
		}
	}
	if proxyUrl != "" {
		p, err := url.Parse(proxyUrl)
		if err != nil {
			logWithErrorf(err, "could not parse proxy URL!")
			return err
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

func getPullSecretName(apiReader client.Reader, dynakubeName string, namespaceName string) (string, error) {
	query := kubeobjects.NewDynakubeQuery(nil, apiReader, namespaceName).WithContext(context.TODO())
	dynakube, err := query.Get(types.NamespacedName{Namespace: namespaceName, Name: dynakubeName})
	if err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", namespaceName, dynakubeName)
		return "", err
	}

	pullSecretName := dynakubeName + pullSecretSuffix
	if dynakube.Spec.CustomPullSecret != "" {
		pullSecretName = dynakube.Spec.CustomPullSecret
	}

	return pullSecretName, nil
}

func getPullSecret(apiReader client.Reader, pullSecretName string, namespaceName string) (string, error) {
	query := kubeobjects.NewSecretQuery(context.TODO(), nil, apiReader, log)
	secret, err := query.Get(types.NamespacedName{Namespace: namespaceName, Name: pullSecretName})
	if err != nil {
		logWithErrorf(err, "'%s:%s' pull secret is missing", namespaceName, pullSecretName)
		return "", err
	}

	secretBytes, ok := secret.Data[".dockerconfigjson"]
	if !ok {
		logErrorf("token .dockerconfigjson does not exist in secret '%s'", pullSecretName)
		return "", fmt.Errorf("token .dockerconfigjson does not exist in secret '%s'", pullSecretName)
	}

	secretStr := string(secretBytes)
	return secretStr, nil
}

func getOneAgentImageEndpoint(troubleshootCtx *troubleshootContext) (string, error) {
	query := kubeobjects.NewDynakubeQuery(nil, troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(context.TODO())
	dynakube, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakubeName})
	if err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return "", err
	}

	customImage := ""
	imageEndpoint := ""
	version := ""

	//sr = [https://acc27517.dev.dynatracelabs.com/api acc27517.dev.dynatracelabs.com/api]
	//er = [acc27517.dev.dynatracelabs.com/api acc27517.dev.dynatracelabs.com]

	//image = "${api_url#*//}"
	sr := removeSchemaRegex.FindStringSubmatch(dynakube.Spec.APIURL)
	//image = "${image%/*}/linux/oneagent"
	er := removeApiEndpointRegex.FindStringSubmatch(sr[1])
	imageEndpoint = er[1] + "/linux/oneagent"

	if dynakube.ClassicFullStackMode() {
		customImage = dynakube.Spec.OneAgent.ClassicFullStack.Image
		version = dynakube.Spec.OneAgent.ClassicFullStack.Version
	} else if dynakube.CloudNativeFullstackMode() {
		customImage = dynakube.Spec.OneAgent.CloudNativeFullStack.Image
		version = dynakube.Spec.OneAgent.CloudNativeFullStack.Version
	} else if dynakube.HostMonitoringMode() {
		customImage = dynakube.Spec.OneAgent.HostMonitoring.Image
		version = dynakube.Spec.OneAgent.HostMonitoring.Version
	}

	if customImage != "" {
		imageEndpoint = customImage
	} else if version != "" {
		imageEndpoint = imageEndpoint + ":" + version
	}

	logInfof("OneAgent image endpoint '%s'", imageEndpoint)
	return imageEndpoint, nil
}

func getActiveGateImageEndpoint(troubleshootCtx *troubleshootContext) (string, error) {
	query := kubeobjects.NewDynakubeQuery(nil, troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(context.TODO())
	dynakube, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakubeName})
	if err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return "", err
	}

	imageEndpoint := ""

	//image = "${api_url#*//}"
	sr := removeSchemaRegex.FindStringSubmatch(dynakube.Spec.APIURL)
	//image = "${image%/*}/linux/activegate"
	er := removeApiEndpointRegex.FindStringSubmatch(sr[1])
	imageEndpoint = er[1] + "/linux/activegate"

	if dynakube.Spec.ActiveGate.Image != "" {
		imageEndpoint = dynakube.Spec.ActiveGate.Image
	}

	logInfof("ActiveGate image endpoint '%s'", imageEndpoint)
	return imageEndpoint, nil
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
		logErrorf("invalid version of the image")
	}
	return
}
