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

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

func checkImagePullable(apiReader client.Reader, troubleshootContext *TestData) error {
	tslog.SetPrefix("[imagepull ] ")

	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "Selected '%s:%s' Dynakube does not exist", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
		return err
	}

	httpClient := &http.Client{
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
	}
	if err := addProxy(apiReader, troubleshootContext, httpClient); err != nil {
		return err
	}

	if dynakube.NeedsOneAgent() {
		err := checkOneAgentImagePullable(httpClient, apiReader, troubleshootContext)
		if err != nil {
			return err
		}
	}

	if dynakube.NeedsActiveGate() {
		err := checkActiveGateImagePullable(httpClient, apiReader, troubleshootContext)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkOneAgentImagePullable(httpClient *http.Client, apiReader client.Reader, troubleshootContext *TestData) error {
	tslog.NewTestf("checking if OneAgent image is pullable ...")

	pullSecretName, err := getPullSecretName(apiReader, troubleshootContext.dynakubeName, troubleshootContext.namespaceName)
	if err != nil {
		return err
	}

	pullSecret, err := getPullSecret(apiReader, pullSecretName, troubleshootContext.namespaceName)
	if err != nil {
		return err
	}

	dynakubeOneAgentImage, err := getOneAgentImageEndpoint(apiReader, troubleshootContext)
	if err != nil {
		return err
	}

	if err = checkComponentImagePullable(httpClient, "OneAgent", pullSecret, dynakubeOneAgentImage); err != nil {
		return err
	}

	return nil
}

func checkActiveGateImagePullable(httpClient *http.Client, apiReader client.Reader, troubleshootContext *TestData) error {
	tslog.NewTestf("checking if ActiveGate image is pullable ...")

	pullSecretName, err := getPullSecretName(apiReader, troubleshootContext.dynakubeName, troubleshootContext.namespaceName)
	if err != nil {
		return err
	}
	pullSecret, err := getPullSecret(apiReader, pullSecretName, troubleshootContext.namespaceName)
	if err != nil {
		return err
	}

	dynakubeActiveGateImage, err := getActiveGateImageEndpoint(apiReader, troubleshootContext)
	if err != nil {
		return err
	}

	if err = checkComponentImagePullable(httpClient, "ActiveGate", pullSecret, dynakubeActiveGateImage); err != nil {
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
	tslog.Infof("using '%s' on '%s' with version '%s' as %s image", componentImage, componentRegistry, componentVersion, componentName)

	imageWorks := false

	// parse docker config
	var result Auths
	json.Unmarshal([]byte(pullSecret), &result)

	for registry, endpoint := range result.Auths {
		tslog.Infof("checking images for registry '%s'", registry)

		apiToken := base64.StdEncoding.EncodeToString([]byte(endpoint.Username + ":" + endpoint.Password))

		if statusCode, err := connectToDockerRegistry(httpClient, "HEAD", "https://"+registry+"/v2/", "Basic", apiToken); err != nil {
			tslog.Errorf("registry '%s' unreachable", registry)
			continue
		} else {
			if statusCode != 200 {
				tslog.Errorf("registry '%s' unreachable (%d)", registry, statusCode)
				continue
			} else {
				tslog.Infof("registry '%s' is accessible", registry)
			}
		}

		if statusCode, err := connectToDockerRegistry(httpClient, "HEAD", "https://"+registry+"/v2/"+componentImage+"/manifests/"+componentVersion, "Basic", apiToken); err != nil {
			tslog.Errorf("registry '%s' unreachable", registry)
			continue
		} else {
			if statusCode != 200 {
				tslog.Errorf("image '%s' with version '%s' not found on registry '%s'", componentImage, componentVersion, registry)
				continue
			} else {
				tslog.Infof("image '%s' with version '%s' exists on registry '%s", componentImage, componentVersion, registry)
			}
		}

		imageWorks = true
	}

	if imageWorks {
		tslog.Okf("%s image '%s' found", componentName, componentRegistry+"/"+componentImage)
	} else {
		tslog.Errorf("%s image '%s' missing", componentName, componentRegistry+"/"+componentImage)
		return fmt.Errorf("%s image '%s' missing", componentName, componentRegistry+"/"+componentImage)
	}
	return nil
}

func addProxy(apiReader client.Reader, troubleshootContext *TestData, httpClient *http.Client) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
		return err
	}

	proxyUrl := ""
	if dynakube.Spec.Proxy.Value != "" {
		proxyUrl = dynakube.Spec.Proxy.Value
	} else if dynakube.Spec.Proxy.ValueFrom != "" {
		proxySecret := corev1.Secret{}
		if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: dynakube.Spec.Proxy.ValueFrom, Namespace: troubleshootContext.namespaceName}, &proxySecret); err != nil {
			tslog.WithErrorf(err, "'%s:%s' proxy secret is missing", dynakube.Spec.Proxy.ValueFrom, troubleshootContext.dynatraceApiSecretName)
			return err
		}
		var err error
		proxyUrl, err = kubeobjects.ExtractToken(&proxySecret, dtclient.CustomProxySecretKey)
		if err != nil {
			tslog.WithErrorf(err, "failed to extract proxy secret field: %w", err)
			return fmt.Errorf("failed to extract proxy secret field: %w", err)
		}
	}

	if proxyUrl != "" {
		p, err := url.Parse(proxyUrl)
		if err != nil {
			tslog.WithErrorf(err, "could not parse proxy URL!")
			return err
		}
		t := httpClient.Transport.(*http.Transport)
		t.Proxy = http.ProxyURL(p)
	}
	return nil
}

func connectToDockerRegistry(httpClient *http.Client, httpMethod string, httpUrl string, authMethod string, authToken string) (int, error) {
	t := httpClient.Transport.(*http.Transport)
	if t.Proxy != nil {
		tslog.Infof("using proxy to connect to the registry")
	}

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
	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: dynakubeName, Namespace: namespaceName}, &dynakube); err != nil {
		tslog.Errorf("selected Dynakube does not exist '%s'", dynakubeName)
		return "", err
	}

	pullSecretName := dynakubeName + pullSecretSuffix
	if dynakube.Spec.CustomPullSecret != "" {
		pullSecretName = dynakube.Spec.CustomPullSecret
	}

	return pullSecretName, nil
}

func getPullSecret(apiReader client.Reader, pullSecretName string, namespaceName string) (string, error) {
	secret := corev1.Secret{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: pullSecretName, Namespace: namespaceName}, &secret); err != nil {
		tslog.Errorf("pull secret '%s' is missing (%s)", pullSecretName, err.Error())
		return "", err
	}

	secretBytes, ok := secret.Data[".dockerconfigjson"]
	if !ok {
		tslog.Errorf("token .dockerconfigjson does not exist in secret '%s'", pullSecretName)
		return "", fmt.Errorf("token .dockerconfigjson does not exist in secret '%s'", pullSecretName)
	}

	secretStr := string(secretBytes)
	return secretStr, nil
}

func getOneAgentImageEndpoint(apiReader client.Reader, troubleshootContext *TestData) (string, error) {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
		return "", err
	}

	customImage := ""
	imageEndpoint := ""

	//sr = [https://acc27517.dev.dynatracelabs.com/api acc27517.dev.dynatracelabs.com/api]
	//er = [acc27517.dev.dynatracelabs.com/api acc27517.dev.dynatracelabs.com]

	//image = "${api_url#*//}"
	sr := removeSchemaRegex.FindStringSubmatch(dynakube.Spec.APIURL)
	//fmt.Printf("OA sr %v\n", sr)
	//image = "${image%/*}/linux/oneagent"
	er := removeApiEndpointRegex.FindStringSubmatch(sr[1])
	//fmt.Printf("OA er %v\n", er)
	imageEndpoint = er[1] + "/linux/oneagent"

	if dynakube.ClassicFullStackMode() {
		customImage = dynakube.Spec.OneAgent.ClassicFullStack.Image
	} else if dynakube.CloudNativeFullstackMode() {
		customImage = dynakube.Spec.OneAgent.CloudNativeFullStack.Image
	} else if dynakube.HostMonitoringMode() {
		customImage = dynakube.Spec.OneAgent.HostMonitoring.Image
	}

	if customImage != "" {
		imageEndpoint = customImage
	}

	// TODO Spec.<type>.Version ?!?

	tslog.Infof("OneAgent image endpoint '%s'", imageEndpoint)
	return imageEndpoint, nil
}

func getActiveGateImageEndpoint(apiReader client.Reader, troubleshootContext *TestData) (string, error) {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
		return "", errors.WithStack(err)
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

	// TODO Spec.<type>.Version ?!?

	tslog.Infof("ActiveGate image endpoint '%s'", imageEndpoint)
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
		tslog.Infof("using latest image version")
	} else if len(fields) >= 2 {
		image = fields[0]
		version = fields[1]
		tslog.Infof("using custom image version")
	} else {
		tslog.Errorf("invalid version of the image")
	}
	return
}
