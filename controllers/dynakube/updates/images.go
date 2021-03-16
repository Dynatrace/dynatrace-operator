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
func ReconcileImageVersions(
	ctx context.Context,
	rec *utils.Reconciliation,
	cl client.Client,
	updateActiveGate bool,
	verProvider VersionProviderCallback,
) (bool, error) {
	dk := rec.Instance

	needsActiveGateUpdate := updateActiveGate &&
		dk.NeedsActiveGate() &&
		rec.IsOutdated(dk.Status.ActiveGate.LastImageProbeTimestamp, ProbeThreshold)

	needsOneAgentUpdate := dk.NeedsImmutableOneAgent() &&
		rec.IsOutdated(dk.Status.OneAgent.LastImageProbeTimestamp, ProbeThreshold) &&
		dk.ShouldAutoUpdateOneAgent()

	if !needsActiveGateUpdate && !needsOneAgentUpdate {
		return false, nil
	}

	var ps corev1.Secret
	if err := cl.Get(ctx, client.ObjectKey{Name: dk.PullSecret(), Namespace: dk.Namespace}, &ps); err != nil {
		return false, errors.WithMessage(err, "failed to get image pull secret")
	}

	auths, err := dtversion.ParseDockerAuthsFromSecret(&ps)
	if err != nil {
		return false, errors.WithMessage(err, "failed to get Dockerconfig for pull secret")
	}

	dockerCfg := dtversion.DockerConfig{Auths: auths, SkipCertCheck: dk.Spec.SkipCertCheck}

	if needsActiveGateUpdate {
		if err := updateImageVersion(rec, dk.ActiveGateImage(), &dk.Status.ActiveGate.ImageStatus, &dockerCfg, verProvider, true); err != nil {
			rec.Log.Error(err, "Failed to update ActiveGate image version")
		}
	}

	if needsOneAgentUpdate {
		if err := updateImageVersion(rec, dk.ImmutableOneAgentImage(), &dk.Status.OneAgent.ImageStatus, &dockerCfg, verProvider, false); err != nil {
			rec.Log.Error(err, "Failed to update OneAgent image version")
		}
	}

	return true, nil
}

func updateImageVersion(
	rec *utils.Reconciliation,
	img string,
	target *dynatracev1alpha1.ImageStatus,
	dockerCfg *dtversion.DockerConfig,
	verProvider VersionProviderCallback,
	allowDowngrades bool,
) error {
	target.LastImageProbeTimestamp = rec.Now.DeepCopy()

	ver, err := verProvider(img, dockerCfg)
	if err != nil {
		return errors.WithMessage(err, "failed to get image version")
	}

	if target.ImageVersion == ver.Version {
		return nil
	}

	if !allowDowngrades && target.ImageVersion != "" {
		oldVer, err := dtversion.ExtractVersion(target.ImageVersion)
		if err != nil {
			return errors.WithMessage(err, "failed to parse old image version")
		}

		newVer, err := dtversion.ExtractVersion(ver.Version)
		if err != nil {
			return errors.WithMessage(err, "failed to parse new image version")
		}

		if dtversion.CompareVersionInfo(oldVer, newVer) > 0 {
			return errors.Errorf("trying to downgrade from '%s' to '%s'", oldVer, newVer)
		}
	}

	rec.Log.Info("Update found",
		"image", img,
		"oldVersion", target.ImageVersion, "newVersion", ver.Version,
		"oldHash", target.ImageHash, "newHash", ver.Hash)

	target.ImageVersion = ver.Version
	target.ImageHash = ver.Hash
	return nil
}
