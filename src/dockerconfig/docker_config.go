package dockerconfig

import (
	"context"
	"encoding/json"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DockerConfig struct {
	ApiReader        client.Reader
	Dynakube         *dynatracev1beta1.DynaKube
	Auths            map[string]DockerAuth
	TrustedCertsPath string
}

type DockerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewDockerConfig(apiReader client.Reader, dynakube dynatracev1beta1.DynaKube) *DockerConfig {
	dockerConfig := DockerConfig{
		ApiReader: apiReader,
		Auths:     make(map[string]DockerAuth),
		Dynakube:  &dynakube,
	}
	return &dockerConfig
}

func (config *DockerConfig) SetupAuths(ctx context.Context) error {
	var pullSecret corev1.Secret
	err := config.ApiReader.Get(ctx, client.ObjectKey{Name: config.Dynakube.PullSecret(), Namespace: config.Dynakube.Namespace}, &pullSecret)
	if err != nil {
		log.Info("failed to load pull secret", "dynakube", config.Dynakube.Name)
		return errors.WithStack(err)
	}
	dockerAuths, err := parseDockerAuthsFromSecret(&pullSecret)
	if err != nil {
		log.Info("failed to parse pull secret content", "dynakube", config.Dynakube.Name)
		return err
	}
	config.Auths = dockerAuths
	return nil
}

func (config *DockerConfig) SaveCustomCAs(
	ctx context.Context,
	fs afero.Afero,
	path string,
) error {
	certs := &corev1.ConfigMap{}
	if err := config.ApiReader.Get(ctx, client.ObjectKey{Namespace: config.Dynakube.Namespace, Name: config.Dynakube.Spec.TrustedCAs}, certs); err != nil {
		log.Info("failed to load trusted CAs")
		return errors.WithStack(err)
	}
	if certs.Data[dtclient.CustomCertificatesConfigMapKey] == "" {
		return errors.New("failed to extract certificate configmap field: missing field certs")
	}
	if err := fs.WriteFile(path, []byte(certs.Data[dtclient.CustomCertificatesConfigMapKey]), 0666); err != nil {
		log.Info("failed to save custom certificates")
		return errors.WithStack(err)
	}
	config.TrustedCertsPath = path
	return nil
}

func (config DockerConfig) SkipCertCheck() bool {
	if config.Dynakube == nil {
		return false
	}
	return config.Dynakube.Spec.SkipCertCheck
}

func parseDockerAuthsFromSecret(secret *corev1.Secret) (map[string]DockerAuth, error) {
	if secret == nil {
		return nil, errors.New("pull secret is nil, parsing not possible")
	}

	config, hasConfig := secret.Data[".dockerconfigjson"]
	if !hasConfig {
		return nil, errors.New("could not find any docker config in image pull secret")
	}

	var dockerConf struct {
		Auths map[string]DockerAuth `json:"auths"`
	}
	err := json.Unmarshal(config, &dockerConf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dockerConf.Auths, nil
}
