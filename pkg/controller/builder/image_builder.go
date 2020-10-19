package builder

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateSecretIfNotExists creates a secret in case it does not exist or updates it if there are changes
func CreateOrUpdateSecretIfNotExists(c client.Client, r client.Reader, secretName string, targetNS string, data map[string][]byte, secretType corev1.SecretType, log logr.Logger) error {
	var cfg corev1.Secret
	err := r.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: targetNS}, &cfg)
	if k8serrors.IsNotFound(err) {
		log.Info("Creating OneAgent config secret")
		if err := c.Create(context.TODO(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: targetNS,
			},
			Type: secretType,
			Data: data,
		}); err != nil {
			return fmt.Errorf("failed to create secret %s: %w", secretName, err)
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to query for secret %s: %w", secretName, err)
	}

	if !reflect.DeepEqual(data, cfg.Data) {
		log.Info(fmt.Sprintf("Updating secret %s", secretName))
		cfg.Data = data
		if err := c.Update(context.TODO(), &cfg); err != nil {
			return fmt.Errorf("failed to update secret %s: %w", secretName, err)
		}
	}

	return nil
}

// GeneratePullSecretData generates the secret data for the PullSecret
func GeneratePullSecretData(instance dynatracev1alpha1.ActiveGate, dtc dtclient.Client) (map[string][]byte, error) {
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

	a := fmt.Sprintf("%s:%s", ci.TenantUUID, _const.DynatracePaasToken)
	a = b64.StdEncoding.EncodeToString([]byte(a))

	auth := auths{
		Username: ci.TenantUUID,
		Password: _const.DynatracePaasToken,
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

func BuildActiveGateImage(apiURL string) (string, error) {
	registry, err := getImageRegistryFromAPIURL(apiURL)
	if err != nil {
		return "", err
	}

	image := registry + "/linux/activegate"

	return image, nil
}

func getImageRegistryFromAPIURL(apiURL string) (string, error) {
	r := strings.TrimPrefix(apiURL, "https://")
	r = strings.TrimSuffix(r, "/api")
	return r, nil
}
