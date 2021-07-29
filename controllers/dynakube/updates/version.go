package updates

import (
	"context"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
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
	rec *utils.Reconciliation,
	cl client.Client,
	verProvider VersionProviderCallback,
) (bool, error) {
	upd := false
	dk := rec.Instance

	needsOneAgentUpdate := dk.NeedsOneAgent() &&
		rec.IsOutdated(dk.Status.OneAgent.LastUpdateProbeTimestamp, ProbeThreshold) &&
		dk.ShouldAutoUpdateOneAgent()

	if needsOneAgentUpdate && !dk.NeedsImmutableOneAgent() {
		upd = true
		if err := updateOneAgentInstallerVersion(rec, dk); err != nil {
			rec.Log.Error(err, "Failed to fetch OneAgent installer version")
		}
	}

	needsActiveGateUpdate := dk.NeedsActiveGate() &&
		!dk.FeatureDisableActiveGateUpdates() &&
		rec.IsOutdated(dk.Status.ActiveGate.LastUpdateProbeTimestamp, ProbeThreshold)

	needsImmutableOneAgentUpdate := dk.NeedsImmutableOneAgent() && needsOneAgentUpdate

	if !needsActiveGateUpdate && !needsImmutableOneAgentUpdate {
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
		if err := updateImageVersion(rec, dk.ActiveGateImage(), &dk.Status.ActiveGate.VersionStatus, &dockerCfg, verProvider, true); err != nil {
			rec.Log.Error(err, "Failed to update ActiveGate image version")
		}
	}

	if needsImmutableOneAgentUpdate {
		if err := updateImageVersion(rec, dk.ImmutableOneAgentImage(), &dk.Status.OneAgent.VersionStatus, &dockerCfg, verProvider, false); err != nil {
			rec.Log.Error(err, "Failed to update OneAgent image version")
		}
	}

	return upd, nil
}

func updateImageVersion(
	rec *utils.Reconciliation,
	img string,
	target *dynatracev1alpha1.VersionStatus,
	dockerCfg *dtversion.DockerConfig,
	verProvider VersionProviderCallback,
	allowDowngrades bool,
) error {
	target.LastUpdateProbeTimestamp = rec.Now.DeepCopy()

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

	rec.Log.Info("Update found",
		"image", img,
		"oldVersion", target.Version, "newVersion", ver.Version,
		"oldHash", target.ImageHash, "newHash", ver.Hash)
	target.Version = ver.Version
	target.ImageHash = ver.Hash
	return nil
}

func updateOneAgentInstallerVersion(rec *utils.Reconciliation, dk *dynatracev1alpha1.DynaKube) error {
	dk.Status.OneAgent.LastUpdateProbeTimestamp = rec.Now.DeepCopy()
	ver := dk.Status.LatestAgentVersionUnixDefault

	oldVer := dk.Status.OneAgent.Version

	if oldVer == ver {
		return nil
	}

	if oldVer != "" {
		if upgrade, err := dtversion.NeedsUpgradeRaw(oldVer, ver); err != nil {
			return err
		} else if !upgrade {
			return nil
		}
	}

	rec.Log.Info("OneAgent update found", "oldVersion", oldVer, "newVersion", ver)
	dk.Status.OneAgent.Version = ver
	return nil
}
