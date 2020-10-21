package builder

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	corev1 "k8s.io/api/core/v1"
)

// GeneratePullSecretData generates the secret data for the PullSecret
func GeneratePullSecretData(instance *dynatracev1alpha1.DynaKube, dtc dtclient.Client, tkns *corev1.Secret) (map[string][]byte, error) {
	type auths struct {
		Username string
		Password string
		Auth     string
	}

	type dockercfg struct {
		Auths map[string]auths
	}

	ci, err := dtc.GetConnectionInfo()
	if err != nil {
		return nil, err
	}

	r, err := getImageRegistryFromAPIURL(instance.Spec.APIURL)
	if err != nil {
		return nil, err
	}

	a := fmt.Sprintf("%s:%s", ci.TenantUUID, string(tkns.Data[_const.DynatracePaasToken]))
	a = b64.StdEncoding.EncodeToString([]byte(a))

	auth := auths{
		Username: ci.TenantUUID,
		Password: string(tkns.Data[_const.DynatracePaasToken]),
		Auth:     a,
	}

	d := dockercfg{
		Auths: map[string]auths{
			r: auth,
		},
	}
	j, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{".dockerconfigjson": j}, nil
}

func BuildActiveGateImage(apiURL string, activegateVersion string) (string, error) {
	registry, err := getImageRegistryFromAPIURL(apiURL)
	if err != nil {
		return "", err
	}

	image := registry + "/linux/activegate"

	if activegateVersion != "" {
		image += ":" + activegateVersion
	}

	return image, nil
}

func getImageRegistryFromAPIURL(apiURL string) (string, error) {
	r := strings.TrimPrefix(apiURL, "https://")
	r = strings.TrimSuffix(r, "/api")
	return r, nil
}
