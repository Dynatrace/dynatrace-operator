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

type Reconciler struct {
	Dynakube        *dynatracev1beta1.DynaKube
	ApiReader       client.Reader
	Fs              afero.Afero
	VersionProvider VersionProviderCallback
	TimeProvider    *kubeobjects.TimeProvider
}

type updateSpec struct {
	updateActiveGate bool
	updateEec        bool
	statsdUpdate     bool
	oneAgentUpdate   bool
}

func (s updateSpec) needsUpdate() bool {
	return s.updateActiveGate || s.updateEec || s.statsdUpdate || s.oneAgentUpdate
}

// Reconcile updates the version and hash for the images used by the rec.Dynakube DynaKube instance.
func (r *Reconciler) Reconcile(ctx context.Context) error {
	updateSpec := updateSpec{
		updateActiveGate: needsActiveGateUpdate(r.Dynakube, *r.TimeProvider),
		updateEec:        needsEecUpdate(r.Dynakube, *r.TimeProvider),
		statsdUpdate:     needsStatsdUpdate(r.Dynakube, *r.TimeProvider),
		oneAgentUpdate:   needsOneAgentUpdate(r.Dynakube, *r.TimeProvider),
	}

	if !updateSpec.needsUpdate() {
		return nil
	}

	return r.updateImages(ctx, updateSpec)
}

func (r *Reconciler) updateImages(ctx context.Context, spec updateSpec) error {
	dockerConfig, err := createDockerConfigWithCustomCAs(ctx, r.Dynakube, r.ApiReader, r.Fs)
	if err != nil {
		return err
	}

	imageUpdater := imageUpdater{
		now:         *r.TimeProvider.Now(),
		dockerCfg:   dockerConfig,
		verProvider: r.VersionProvider,
	}
	if spec.updateActiveGate {
		err := imageUpdater.update(r.Dynakube.ActiveGateImage(), &r.Dynakube.Status.ActiveGate.VersionStatus, true)
		if err != nil {
			log.Error(err, "failed to update ActiveGate image version")
		}
	}

	if spec.updateEec {
		err := imageUpdater.update(r.Dynakube.EecImage(), &r.Dynakube.Status.ExtensionController.VersionStatus, true)
		if err != nil {
			log.Error(err, "Failed to update Extension Controller image version")
		}
	}

	if spec.statsdUpdate {
		err := imageUpdater.update(r.Dynakube.StatsdImage(), &r.Dynakube.Status.Statsd.VersionStatus, true)
		if err != nil {
			log.Error(err, "Failed to update StatsD image version")
		}
	}

	if spec.oneAgentUpdate {
		err := imageUpdater.update(r.Dynakube.OneAgentImage(), &r.Dynakube.Status.OneAgent.VersionStatus, false)
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

type imageUpdater struct {
	now         metav1.Time
	dockerCfg   *dockerconfig.DockerConfig
	verProvider VersionProviderCallback
}

func (updater imageUpdater) update(
	img string,
	target *dynatracev1beta1.VersionStatus,
	allowDowngrades bool,
) error {
	target.LastUpdateProbeTimestamp = &updater.now

	ver, err := updater.verProvider(img, updater.dockerCfg)
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
