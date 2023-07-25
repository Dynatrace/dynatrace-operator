package containerregistry

import (
	"sync"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	dockertypes "github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/afero"
)

type dockerKeychain struct {
	mutex            sync.Mutex
	dockerConfigFile string
	filesystem       afero.Fs
}

func NewDockerKeychain(dockerconfigFile string, filesystem afero.Fs) authn.Keychain {
	return &dockerKeychain{
		dockerConfigFile: dockerconfigFile,
		filesystem:       filesystem,
	}
}

// Resolve implements Keychain interface by interpreting the docker config file.
// It is based on the 'defaultKeychain' type from the go-gontainerregistry library
// https://github.com/google/go-containerregistry/blob/27a6ad6/pkg/authn/keychain.go
// dockerKeychain implementation can read a docker config file of any name and from any directory.
func (dk *dockerKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	dk.mutex.Lock()
	defer dk.mutex.Unlock()

	var cf *configfile.ConfigFile

	f, err := dk.filesystem.Open(dk.dockerConfigFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cf, err = config.LoadFromReader(f)
	if err != nil {
		return nil, err
	}

	// See:
	// https://github.com/google/ko/issues/90
	// https://github.com/moby/moby/blob/fc01c2b481097a6057bec3cd1ab2d7b4488c50c4/registry/config.go#L397-L404
	var cfg, empty dockertypes.AuthConfig
	for _, key := range []string{
		target.String(),
		target.RegistryStr(),
	} {
		if key == name.DefaultRegistry {
			key = authn.DefaultAuthKey
		}

		cfg, err = cf.GetAuthConfig(key)
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
