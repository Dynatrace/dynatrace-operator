package dockerconfig

import (
	"context"
	"path"
	"path/filepath"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TmpPath         = "/tmp/dynatrace-operator"
	CADir           = "ca"
	TrustedCertFileName      = "custom.crt"
	RegistryAuthDir = "dockerconf"
)

type DockerConfig struct {
	ApiReader          client.Reader
	Dynakube           *dynatracev1beta1.DynaKube
	RegistryAuthPath   string
	TrustedCertsPath   string
	registryAuthSecret *corev1.Secret
}

func NewDockerConfig(apiReader client.Reader, dynakube dynatracev1beta1.DynaKube) *DockerConfig {
	trustedCertsPath := ""
	if dynakube.Spec.TrustedCAs != "" {
		trustedCertsPath = path.Join(TmpPath, CADir, dynakube.Name)
	}
	dockerConfig := DockerConfig{
		ApiReader:        apiReader,
		Dynakube:         &dynakube,
		RegistryAuthPath: path.Join(TmpPath, RegistryAuthDir, dynakube.Name),
		TrustedCertsPath: trustedCertsPath,
	}
	return &dockerConfig
}

func (config *DockerConfig) StoreRequiredFiles(ctx context.Context, fs afero.Afero) error {
	if err := config.storeRegistryCredentials(ctx, fs); err != nil {
		return err
	}

	if config.Dynakube.Spec.TrustedCAs != "" {
		if err := config.storeTrustedCAs(ctx, fs); err != nil {
			return err
		}
	}

	return nil
}

func (config *DockerConfig) SetRegistryAuthSecret(secret *corev1.Secret) {
	config.registryAuthSecret = secret
}

func (config *DockerConfig) SkipCertCheck() bool {
	if config.Dynakube == nil {
		return false
	}
	return config.Dynakube.Spec.SkipCertCheck
}

func (config *DockerConfig) Cleanup(fs afero.Afero) error {
	if err := fs.RemoveAll(config.RegistryAuthPath); err != nil {
		log.Info("failed to remove registry credentials", "dynakube", config.Dynakube.Name, "error", err)
	}
	if err := fs.RemoveAll(config.TrustedCertsPath); err != nil {
		log.Info("failed to remove custom certificates", "dynakube", config.Dynakube.Name, "error", err)
	}
	return nil
}

func (config *DockerConfig) getRegistryCredentials(ctx context.Context) ([]byte, error) {
	var pullSecret corev1.Secret
	if config.registryAuthSecret != nil {
		pullSecret = *config.registryAuthSecret
	} else {
		if err := config.ApiReader.Get(ctx, client.ObjectKey{Namespace: config.Dynakube.Namespace, Name: config.Dynakube.PullSecret()}, &pullSecret); err != nil {
			log.Info("failed to load registry pull secret")
			return nil, errors.WithStack(err)
		}
	}
	dockerAuths, err := extractDockerAuthsFromSecret(&pullSecret)
	if err != nil {
		log.Info("failed to parse pull secret content", "dynakube", config.Dynakube.Name)
		return nil, err
	}
	return dockerAuths, nil
}

func (config *DockerConfig) storeRegistryCredentials(ctx context.Context, fs afero.Afero) error {
	registryCredentials, err := config.getRegistryCredentials(ctx)
	if err != nil {
		return err
	}

	if err := saveFile(registryCredentials, fs, config.RegistryAuthPath); err != nil {
		log.Info("failed to store registry credentials", "dynakube", config.Dynakube.Name)
		return err
	}

	return nil
}

func (config *DockerConfig) storeTrustedCAs(ctx context.Context, fs afero.Afero) error {
	customCAs, err := config.Dynakube.TrustedCAs(ctx, config.ApiReader)
	if err != nil {
		return err
	}

	tmpCertFilePath := path.Join(config.TrustedCertsPath, TrustedCertFileName)
	if err := saveFile(customCAs, fs, tmpCertFilePath); err != nil {
		log.Info("failed to store custom certificates", "dynakube", config.Dynakube.Name)
		return err
	}

	return nil
}

func saveFile(data []byte, fs afero.Afero, path string) error {
	if err := fs.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return fs.WriteFile(path, data, 0666)
}

func extractDockerAuthsFromSecret(secret *corev1.Secret) ([]byte, error) {
	if secret == nil {
		return nil, errors.New("pull secret is nil, parsing not possible")
	}

	config, hasConfig := secret.Data[".dockerconfigjson"]
	if !hasConfig {
		return nil, errors.New("could not find any docker config in image pull secret")
	}

	return config, nil
}
