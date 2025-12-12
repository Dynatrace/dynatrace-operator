package dockerkeychain

import (
	"bytes"
	"context"
	"sync"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	dockertypes "github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DockerKeychain struct {
	dockerConfig *configfile.ConfigFile
	mutex        sync.Mutex
}

func NewDockerKeychains(ctx context.Context, apiReader client.Reader, namespaceName string, pullSecretNames []string) (authn.Keychain, error) {
	keychain := &DockerKeychain{}

	if len(pullSecretNames) == 0 {
		return keychain, nil
	}

	configFile := configfile.ConfigFile{
		AuthConfigs: make(map[string]dockertypes.AuthConfig),
	}

	for _, pullSecretName := range pullSecretNames {
		pullSecret := corev1.Secret{}

		if err := apiReader.Get(ctx, client.ObjectKey{Namespace: namespaceName, Name: pullSecretName}, &pullSecret); err != nil {
			log.Info("registry pull secret not loaded", "name", pullSecretName, "namespace", namespaceName)

			continue
		}

		dockerAuths, err := extractDockerAuthsFromSecret(&pullSecret)
		if err != nil {
			log.Info("failed to parse pull secret content", "name", pullSecret.Name, "namespace", pullSecret.Namespace)

			return keychain, err
		}

		err = configFile.LoadFromReader(bytes.NewReader(dockerAuths))
		if err != nil {
			return keychain, errors.WithStack(err)
		}
	}

	if len(configFile.AuthConfigs) > 0 {
		keychain.dockerConfig = &configFile
		log.Debug("loaded docker configs", "registries", maps.Keys(configFile.AuthConfigs))
	} else {
		log.Debug("no docker configs found")
	}

	return keychain, nil
}

func NewDockerKeychain(ctx context.Context, apiReader client.Reader, pullSecret corev1.Secret) (authn.Keychain, error) {
	keychain := &DockerKeychain{}

	if pullSecret.Name == "" {
		return keychain, nil
	}

	if err := apiReader.Get(ctx, client.ObjectKey{Namespace: pullSecret.Namespace, Name: pullSecret.Name}, &pullSecret); err != nil {
		log.Info("No registry pull secret loaded", "name", pullSecret.Name, "namespace", pullSecret.Namespace, "err", err)

		return keychain, nil
	}

	dockerAuths, err := extractDockerAuthsFromSecret(&pullSecret)
	if err != nil {
		log.Info("failed to parse pull secret content", "name", pullSecret.Name, "namespace", pullSecret.Namespace)

		return keychain, err
	}

	cf, err := config.LoadFromReader(bytes.NewReader(dockerAuths))
	if err != nil {
		return keychain, errors.WithStack(err)
	}

	keychain.dockerConfig = cf

	return keychain, nil
}

func extractDockerAuthsFromSecret(secret *corev1.Secret) ([]byte, error) {
	if secret == nil {
		return nil, errors.New("pull secret is nil, parsing not possible")
	}

	cfg, hasConfig := secret.Data[".dockerconfigjson"]
	if !hasConfig {
		return nil, errors.New("could not find any docker config in image pull secret")
	}

	return cfg, nil
}

// Resolve implements Keychain interface by interpreting the docker config file.
// It is based on the 'defaultKeychain' type from the go-gontainerregistry library
// https://github.com/google/go-containerregistry/blob/27a6ad6/pkg/authn/keychain.go
// DockerKeychain implementation can read a docker config file of any name and from any directory.
func (keychain *DockerKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	keychain.mutex.Lock()
	defer keychain.mutex.Unlock()

	if keychain.dockerConfig == nil {
		return authn.Anonymous, nil
	}

	// See:
	// https://github.com/google/ko/issues/90
	// https://github.com/moby/moby/blob/fc01c2b481097a6057bec3cd1ab2d7b4488c50c4/registry/config.go#L397-L404
	var cfg, empty dockertypes.AuthConfig

	var err error

	for _, key := range []string{
		target.String(),
		target.RegistryStr(),
	} {
		if key == name.DefaultRegistry {
			key = authn.DefaultAuthKey
		}

		cfg, err = keychain.dockerConfig.GetAuthConfig(key)
		if err != nil {
			return nil, err
		}
		// cf.GetAuthConfig automatically sets the ServerAddress attribute. Since
		// we don't make use of it, clear the value for a proper "is-empty" test.
		// See: https://github.com/google/go-containerregistry/issues/1510
		cfg.ServerAddress = ""
		if cfg != empty {
			break
		}
	}

	if cfg == empty {
		return authn.Anonymous, nil
	}

	return authn.FromConfig(authn.AuthConfig{
		Username:      cfg.Username,
		Password:      cfg.Password,
		Auth:          cfg.Auth,
		IdentityToken: cfg.IdentityToken,
		RegistryToken: cfg.RegistryToken,
	}), nil
}
