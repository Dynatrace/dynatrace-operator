package keychain

import (
	"os"
	"sync"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	dockertypes "github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

// doKeychain implements Keychain with the semantics of the standard Docker
// credential keychain.
type DoKeychain struct {
	mu               sync.Mutex
	dockerConfigFile string
}

func NewDoKeychain(dockerconfigFile string) DoKeychain {
	return DoKeychain{
		dockerConfigFile: dockerconfigFile,
	}
}

// Resolve implements Keychain.
func (dk *DoKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	dk.mu.Lock()
	defer dk.mu.Unlock()

	var cf *configfile.ConfigFile

	f, err := os.Open(dk.dockerConfigFile)
	if err != nil {
		return authn.Anonymous, nil //nolint
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
