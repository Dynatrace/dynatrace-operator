package version

import (
	"context"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultHealthConfigInterval    = 10 * time.Second
	DefaultHealthConfigStartPeriod = 1200 * time.Second
	DefaultHealthConfigTimeout     = 30 * time.Second
	DefaultHealthConfigRetries     = 3
)

// Constructor setting default values for docker image HealthConfig.
func newHealthConfig() *containerv1.HealthConfig {
	return &containerv1.HealthConfig{
		Test:        []string{},
		Interval:    DefaultHealthConfigInterval,
		StartPeriod: DefaultHealthConfigStartPeriod,
		Timeout:     DefaultHealthConfigTimeout,
		Retries:     DefaultHealthConfigRetries,
	}
}

func GetOneAgentHealthConfig(ctx context.Context, apiReader client.Reader, registryClient registry.ImageGetter, dynakube *dynatracev1beta1.DynaKube, imageUri string) (*containerv1.HealthConfig, error) {
	imageInfo, err := registryClient.PullImageInfo(ctx, imageUri)
	if err != nil {
		return nil, errors.WithMessage(err, "error pulling image info")
	}

	configFile, err := (*imageInfo).ConfigFile()
	if err != nil {
		return nil, errors.WithMessage(err, "error reading image config file")
	}

	var healthConfig *containerv1.HealthConfig

	// Healthcheck.Test values from go-containerregistry documentation:
	// {} : inherit healthcheck
	// {"NONE"} : disable healthcheck
	// {"CMD", args...} : exec arguments directly
	// {"CMD-SHELL", command} : run command with system's default shell
	if configFile.Config.Healthcheck != nil && len(configFile.Config.Healthcheck.Test) > 0 {
		var testCommand []string

		switch configFile.Config.Healthcheck.Test[0] {
		case "CMD-SHELL":
			testCommand = []string{"/bin/sh", "-c"}
			testCommand = append(testCommand, configFile.Config.Healthcheck.Test[1:]...)
		case "CMD":
			testCommand = configFile.Config.Healthcheck.Test[1:]
		}

		if len(testCommand) > 0 {
			healthConfig = newHealthConfig()
			healthConfig.Test = testCommand
			if configFile.Config.Healthcheck.Interval != 0 {
				healthConfig.Interval = configFile.Config.Healthcheck.Interval
			}
			if configFile.Config.Healthcheck.StartPeriod != 0 {
				healthConfig.StartPeriod = configFile.Config.Healthcheck.StartPeriod
			}
			if configFile.Config.Healthcheck.Timeout != 0 {
				healthConfig.Timeout = configFile.Config.Healthcheck.Timeout
			}
			if configFile.Config.Healthcheck.Retries != 0 {
				healthConfig.Retries = configFile.Config.Healthcheck.Retries
			}
		}
	}
	return healthConfig, nil
}
