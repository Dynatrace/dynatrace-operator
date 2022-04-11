package dockerconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DockerConfig struct {
	Auths            map[string]DockerAuth
	SkipCertCheck    bool
	TrustedCertsPath string
}

type DockerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewDockerConfig(ctx context.Context, kubeClient client.Reader, dynakube dynatracev1beta1.DynaKube) (*DockerConfig, error) {
	var pullSecret corev1.Secret
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: dynakube.PullSecret(), Namespace: dynakube.Namespace}, &pullSecret); err != nil {
		log.Info("failed to load pull secret", "dynakube", dynakube.Name)
		return nil, err
	}
	dockerAuths, err := parseDockerAuthsFromSecret(&pullSecret)
	if err != nil {
		log.Info("failed to parse pull secret content", "dynakube", dynakube.Name)
		return nil, err
	}

	dockerConfig := DockerConfig{
		Auths:         dockerAuths,
		SkipCertCheck: dynakube.Spec.SkipCertCheck,
	}
	return &dockerConfig, nil

}

func parseDockerAuthsFromSecret(secret *corev1.Secret) (map[string]DockerAuth, error) {
	if secret == nil {
		return nil, fmt.Errorf("given secret is nil")
	}

	config, hasConfig := secret.Data[".dockerconfigjson"]
	if !hasConfig {
		return nil, fmt.Errorf("could not find any docker config in image pull secret")
	}

	var dockerConf struct {
		Auths map[string]DockerAuth `json:"auths"`
	}
	err := json.Unmarshal(config, &dockerConf)
	if err != nil {
		return nil, err
	}

	return dockerConf.Auths, nil
}

func (config *DockerConfig) SaveCustomCAs(ctx context.Context, kubeClient client.Reader, dynakube dynatracev1beta1.DynaKube, path string) error {
	certs := &corev1.ConfigMap{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Namespace: dynakube.Namespace, Name: dynakube.Spec.TrustedCAs}, certs); err != nil {
		log.Info("failed to load trusted CAs")
		return err
	}
	if certs.Data[dtclient.CustomCertificatesConfigMapKey] == "" {
		return fmt.Errorf("failed to extract certificate configmap field: missing field certs")
	}
	if err := os.WriteFile(path, []byte(certs.Data[dtclient.CustomCertificatesConfigMapKey]), 0666); err != nil {
		log.Info("failed to save custom certificates")
		return err
	}
	config.TrustedCertsPath = path
	return nil
}
