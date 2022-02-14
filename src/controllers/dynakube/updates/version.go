package updates

import (
	"context"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtversion"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProbeThreshold is the minimum time to wait between version upgrades.
const ProbeThreshold = 15 * time.Minute

// VersionProviderCallback fetches the version for a given image.
type VersionProviderCallback func(string, *dtversion.DockerConfig) (dtversion.ImageVersion, error)

// ReconcileImageVersions updates the version and hash for the images used by the rec.Instance DynaKube instance.
func ReconcileVersions(
	ctx context.Context,
	dkState *status.DynakubeState,
	cl client.Client,
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
		return upd, nil
	}

	var ps corev1.Secret
	if err := cl.Get(ctx, client.ObjectKey{Name: dk.PullSecret(), Namespace: dk.Namespace}, &ps); err != nil {
		return upd, errors.WithMessage(err, "failed to get image pull secret")
	}

	auths, err := dtversion.ParseDockerAuthsFromSecret(&ps)
	if err != nil {
		return upd, errors.WithMessage(err, "failed to get Dockerconfig for pull secret")
	}

	dockerCfg := dtversion.DockerConfig{Auths: auths, SkipCertCheck: dk.Spec.SkipCertCheck}
	upd = true // updateImageVersion() always updates the status

	if needsActiveGateUpdate {
		if err := updateImageVersion(dkState, dk.ActiveGateImage(), &dk.Status.ActiveGate.VersionStatus, &dockerCfg, verProvider, true); err != nil {
			log.Error(err, "failed to update ActiveGate image version")
		}
	}

	if needsEecUpdate {
		if err := updateImageVersion(dkState, dk.EecImage(), &dk.Status.ExtensionController.VersionStatus, &dockerCfg, verProvider, true); err != nil {
			log.Error(err, "Failed to update Extension Controller image version")
		}
	}

	if needsStatsdUpdate {
		if err := updateImageVersion(dkState, dk.StatsdImage(), &dk.Status.Statsd.VersionStatus, &dockerCfg, verProvider, true); err != nil {
			log.Error(err, "Failed to update StatsD image version")
		}
	}

	if needsOneAgentUpdate {
		if err := updateImageVersion(dkState, dk.ImmutableOneAgentImage(), &dk.Status.OneAgent.VersionStatus, &dockerCfg, verProvider, false); err != nil {
			log.Error(err, "failed to update OneAgent image version")
		}
	}

	return upd, nil
}

func updateImageVersion(
	dkState *status.DynakubeState,
	img string,
	target *dynatracev1beta1.VersionStatus,
	dockerCfg *dtversion.DockerConfig,
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
		if upgrade, err := dtversion.NeedsUpgradeRaw(target.Version, ver.Version); err != nil {
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
