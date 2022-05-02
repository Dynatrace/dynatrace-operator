package version

import (
	"context"
	"os"
	"path"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ProbeThreshold is the minimum time to wait between version upgrades.
	ProbeThreshold = 15 * time.Minute

	TmpCAPath = "/tmp/dynatrace-operator"
	TmpCAName = "dynatraceCustomCA.crt"
)

// VersionProviderCallback fetches the version for a given image.
type VersionProviderCallback func(string, *dockerconfig.DockerConfig) (ImageVersion, error)

// ReconcileVersions updates the version and hash for the images used by the rec.Instance DynaKube instance.
func ReconcileVersions(
	ctx context.Context,
	dkState *status.DynakubeState,
	apiReader client.Reader,
	fs afero.Afero,
	verProvider VersionProviderCallback,
) (bool, error) {
	upd := false
	dk := dkState.Instance

	needsOneAgentUpdate := dk.NeedsOneAgent() &&
		dkState.IsOutdated(dk.Status.OneAgent.LastUpdateProbeTimestamp, ProbeThreshold) &&
		dk.ShouldAutoUpdateOneAgent()

	needsActiveGateUpdate := dk.NeedsActiveGate() &&
		!dk.FeatureDisableActiveGateUpdates() &&
		dkState.IsOutdated(dk.Status.ActiveGate.LastUpdateProbeTimestamp, ProbeThreshold)

	needsEecUpdate := dk.NeedsStatsd() &&
		!dk.FeatureDisableActiveGateUpdates() &&
		dkState.IsOutdated(dk.Status.ExtensionController.LastUpdateProbeTimestamp, ProbeThreshold)

	needsStatsdUpdate := dk.NeedsStatsd() &&
		!dk.FeatureDisableActiveGateUpdates() &&
		dkState.IsOutdated(dk.Status.Statsd.LastUpdateProbeTimestamp, ProbeThreshold)

	if !(needsActiveGateUpdate || needsOneAgentUpdate || needsEecUpdate || needsStatsdUpdate) {
		return false, nil
	}

	caCertPath := path.Join(TmpCAPath, TmpCAName)
	dockerConfig, err := dockerconfig.NewDockerConfig(ctx, apiReader, *dkState.Instance)
	if err != nil {
		return false, err
	}
	if dk.Spec.TrustedCAs != "" {
		_ = os.MkdirAll(TmpCAPath, 0755)
		err := dockerConfig.SaveCustomCAs(ctx, fs, caCertPath)
		if err != nil {
			return false, err
		}
		defer func() {
			_ = os.Remove(TmpCAPath)
		}()
	}
	upd = true // updateImageVersion() always updates the status

	if needsActiveGateUpdate {
		if err := updateImageVersion(dkState, dk.ActiveGateImage(), &dk.Status.ActiveGate.VersionStatus, dockerConfig, verProvider, true); err != nil {
			log.Error(err, "failed to update ActiveGate image version")
		}
	}

	if needsEecUpdate {
		if err := updateImageVersion(dkState, dk.EecImage(), &dk.Status.ExtensionController.VersionStatus, dockerConfig, verProvider, true); err != nil {
			log.Error(err, "Failed to update Extension Controller image version")
		}
	}

	if needsStatsdUpdate {
		if err := updateImageVersion(dkState, dk.StatsdImage(), &dk.Status.Statsd.VersionStatus, dockerConfig, verProvider, true); err != nil {
			log.Error(err, "Failed to update StatsD image version")
		}
	}

	if needsOneAgentUpdate {
		if err := updateImageVersion(dkState, dk.ImmutableOneAgentImage(), &dk.Status.OneAgent.VersionStatus, dockerConfig, verProvider, false); err != nil {
			log.Error(err, "failed to update OneAgent image version")
		}
	}

	return upd, nil
}

func updateImageVersion(
	dkState *status.DynakubeState,
	img string,
	target *dynatracev1beta1.VersionStatus,
	dockerCfg *dockerconfig.DockerConfig,
	verProvider VersionProviderCallback,
	allowDowngrades bool,
) error {
	target.LastUpdateProbeTimestamp = dkState.Now.DeepCopy()

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
