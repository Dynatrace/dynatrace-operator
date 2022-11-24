package version

import (
	"context"
	"os"
	"path"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ProbeThreshold is the minimum time to wait between version upgrades.
	ProbeThreshold = 15 * time.Minute

	TmpCAPath = "/tmp/dynatrace-operator"
	TmpCAName = "dynatraceCustomCA.crt"
)

// VersionProviderCallback fetches the version for a given image.
type VersionProviderCallback func(string, *dockerconfig.DockerConfig) (ImageVersion, error) //nolint:revive

// ReconcileVersions updates the version and hash for the images used by the rec.Dynakube DynaKube instance.
func ReconcileVersions(
	ctx context.Context,
	dynakube *dynatracev1beta1.DynaKube,
	apiReader client.Reader,
	fs afero.Afero,
	versionProvider VersionProviderCallback,
	timeProvider kubeobjects.TimeProvider,
) error {
	needsActiveGateUpdate := needsActiveGateUpdate(dynakube, timeProvider)
	needsEecUpdate := needsEecUpdate(dynakube, timeProvider)
	needsStatsdUpdate := needsStatsdUpdate(dynakube, timeProvider)
	needsOneAgentUpdate := needsOneAgentUpdate(dynakube, timeProvider)

	if !(needsActiveGateUpdate || needsEecUpdate || needsStatsdUpdate || needsOneAgentUpdate) {
		return nil
	}

	return updateImages(ctx, dynakube, apiReader, fs, versionProvider, timeProvider, needsActiveGateUpdate, needsEecUpdate, needsStatsdUpdate, needsOneAgentUpdate)
}

func updateImages(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader, fs afero.Afero, versionProvider VersionProviderCallback, timeProvider kubeobjects.TimeProvider, needsActiveGateUpdate bool, needsEecUpdate bool, needsStatsdUpdate bool, needsOneAgentUpdate bool) error {
	now := timeProvider.Now()
	dockerConfig, err := createDockerConfigWithCustomCAs(ctx, dynakube, apiReader, fs)
	if err != nil {
		return err
	}

	if needsActiveGateUpdate {
		err := updateImageVersion(*now, dynakube.ActiveGateImage(), &dynakube.Status.ActiveGate.VersionStatus, dockerConfig, versionProvider, true)
		if err != nil {
			log.Error(err, "failed to update ActiveGate image version")
		}
	}

	if needsEecUpdate {
		err := updateImageVersion(*now, dynakube.EecImage(), &dynakube.Status.ExtensionController.VersionStatus, dockerConfig, versionProvider, true)
		if err != nil {
			log.Error(err, "Failed to update Extension Controller image version")
		}
	}

	if needsStatsdUpdate {
		err := updateImageVersion(*now, dynakube.StatsdImage(), &dynakube.Status.Statsd.VersionStatus, dockerConfig, versionProvider, true)
		if err != nil {
			log.Error(err, "Failed to update StatsD image version")
		}
	}

	if needsOneAgentUpdate {
		err := updateImageVersion(*now, dynakube.OneAgentImage(), &dynakube.Status.OneAgent.VersionStatus, dockerConfig, versionProvider, false)
		if err != nil {
			log.Error(err, "failed to update OneAgent image version")
		}
	}
	return nil
}

func createDockerConfigWithCustomCAs(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader, fs afero.Afero) (*dockerconfig.DockerConfig, error) {
	caCertPath := path.Join(TmpCAPath, TmpCAName)
	dockerConfig := dockerconfig.NewDockerConfig(apiReader, *dynakube)
	err := dockerConfig.SetupAuths(ctx)
	if err != nil {
		log.Info("failed to set up auths for image version checks")
		return nil, err
	}
	if dynakube.Spec.TrustedCAs != "" {
		_ = os.MkdirAll(TmpCAPath, 0755)
		err := dockerConfig.SaveCustomCAs(ctx, fs, caCertPath)
		if err != nil {
			log.Info("failed to save CAs locally for image version checks")
			return nil, err
		}
		defer func() {
			_ = os.Remove(TmpCAPath)
		}()
	}
	return dockerConfig, nil
}

func needsStatsdUpdate(dynakube *dynatracev1beta1.DynaKube, timeProvider kubeobjects.TimeProvider) bool {
	return dynakube.IsStatsdActiveGateEnabled() &&
		!dynakube.FeatureDisableActiveGateUpdates() &&
		timeProvider.IsOutdated(dynakube.Status.Statsd.LastUpdateProbeTimestamp, ProbeThreshold)
}

func needsEecUpdate(dynakube *dynatracev1beta1.DynaKube, timeProvider kubeobjects.TimeProvider) bool {
	return dynakube.IsStatsdActiveGateEnabled() &&
		!dynakube.FeatureDisableActiveGateUpdates() &&
		timeProvider.IsOutdated(dynakube.Status.ExtensionController.LastUpdateProbeTimestamp, ProbeThreshold)
}

func needsActiveGateUpdate(dynakube *dynatracev1beta1.DynaKube, timeProvider kubeobjects.TimeProvider) bool {
	return dynakube.NeedsActiveGate() &&
		!dynakube.FeatureDisableActiveGateUpdates() &&
		timeProvider.IsOutdated(dynakube.Status.ActiveGate.LastUpdateProbeTimestamp, ProbeThreshold)
}

func needsOneAgentUpdate(dynakube *dynatracev1beta1.DynaKube, timeProvider kubeobjects.TimeProvider) bool {
	return dynakube.NeedsOneAgent() &&
		timeProvider.IsOutdated(dynakube.Status.OneAgent.LastUpdateProbeTimestamp, ProbeThreshold) &&
		dynakube.ShouldAutoUpdateOneAgent()
}

func updateImageVersion(
	now metav1.Time,
	img string,
	target *dynatracev1beta1.VersionStatus,
	dockerCfg *dockerconfig.DockerConfig,
	verProvider VersionProviderCallback,
	allowDowngrades bool,
) error {
	target.LastUpdateProbeTimestamp = &now

	ver, err := verProvider(img, dockerCfg)
	if err != nil {
		return errors.WithMessage(err, "failed to get image version")
	}

	if target.Version == ver.Version {
		return nil
	}

	if !allowDowngrades && target.Version != "" {
		if upgrade, err := version.NeedsUpgradeRaw(target.Version, ver.Version); err != nil {
			return err
		} else if !upgrade {
			return nil
		}
	}

	log.Info("update found",
		"image", img,
		"oldVersion", target.Version, "newVersion", ver.Version,
		"oldHash", target.ImageHash, "newHash", ver.Hash)
	target.Version = ver.Version
	target.ImageHash = ver.Hash
	return nil
}
