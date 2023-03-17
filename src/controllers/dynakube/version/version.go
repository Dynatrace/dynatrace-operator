package version

import (
	"context"
	"fmt"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ProbeThreshold is the minimum time to wait between version upgrades.
	ProbeThreshold = 15 * time.Minute
)

// VersionProviderCallback fetches the version for a given image.
type VersionProviderCallback func(string, *dockerconfig.DockerConfig) (ImageVersion, error) //nolint:revive

type Reconciler struct {
	Dynakube        *dynatracev1beta1.DynaKube
	ApiReader       client.Reader
	Fs              afero.Afero
	VersionProvider VersionProviderCallback
	TimeProvider    *timeprovider.Provider
}

type toUpdateImage struct {
	url             string
	component       string
	version         *dynatracev1beta1.VersionStatus
	allowsDowngrade bool
}

func (i toUpdateImage) update(updater *imageUpdater) {
	err := updater.update(i)
	if err != nil {
		log.Error(err,
			fmt.Sprintf("failed to update %s image version", i.component))
	}
}

// Reconcile updates the version and hash for the images used by the rec.Dynakube DynaKube instance.
func (r *Reconciler) Reconcile(ctx context.Context) error {
	images := []toUpdateImage{}
	addToUpdateImage := func(
		needsUpdate bool,
		image string,
		component string,
		version *dynatracev1beta1.VersionStatus,
		allowsDowngrade bool,
	) {
		if !needsUpdate {
			return
		}

		images = append(images,
			toUpdateImage{
				image,
				component,
				version,
				allowsDowngrade,
			})
	}

	addToUpdateImage(
		r.needsActiveGateUpdate(),
		r.Dynakube.ActiveGateImage(),
		r.Dynakube.Status.ActiveGate.Name(),
		&r.Dynakube.Status.ActiveGate.VersionStatus,
		true)
	addToUpdateImage(
		r.needsOneAgentUpdate(),
		r.Dynakube.OneAgentImage(),
		r.Dynakube.Status.OneAgent.Name(),
		&r.Dynakube.Status.OneAgent.VersionStatus,
		false)
	addToUpdateImage(
		r.needsSynMonitoringUpdate(),
		r.Dynakube.SyntheticImage(),
		r.Dynakube.Status.Synthetic.Name(),
		&r.Dynakube.Status.Synthetic.VersionStatus,
		true)

	if len(images) == 0 {
		return nil
	}

	return r.updateImages(ctx, images)
}

func (r *Reconciler) updateImages(ctx context.Context, images []toUpdateImage) error {
	dockerConfig, err := createDockerConfigWithCustomCAs(ctx, r.Dynakube, r.ApiReader, r.Fs)
	if err != nil {
		return err
	}

	imageUpdater := &imageUpdater{
		now:         *r.TimeProvider.Now(),
		dockerCfg:   dockerConfig,
		verProvider: r.VersionProvider,
	}
	for _, image := range images {
		image.update(imageUpdater)
	}

	return nil
}

func createDockerConfigWithCustomCAs(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader, fs afero.Afero) (*dockerconfig.DockerConfig, error) {
	dockerConfig := dockerconfig.NewDockerConfig(apiReader, *dynakube)
	err := dockerConfig.StoreRequiredFiles(ctx, fs)
	if err != nil {
		log.Info("failed to store required files for docker config")
		return nil, err
	}
	return dockerConfig, nil
}

func (r *Reconciler) needsActiveGateUpdate() bool {
	return r.Dynakube.NeedsActiveGate() &&
		!r.Dynakube.FeatureDisableActiveGateUpdates() &&
		r.TimeProvider.IsOutdated(r.Dynakube.Status.ActiveGate.LastUpdateProbeTimestamp, ProbeThreshold)
}

func (r *Reconciler) needsOneAgentUpdate() bool {
	return r.Dynakube.NeedsOneAgent() &&
		r.TimeProvider.IsOutdated(r.Dynakube.Status.OneAgent.LastUpdateProbeTimestamp, ProbeThreshold) &&
		r.Dynakube.ShouldAutoUpdateOneAgent()
}

func (r *Reconciler) needsSynMonitoringUpdate() bool {
	return r.Dynakube.IsSyntheticMonitoringEnabled() &&
		r.TimeProvider.IsOutdated(r.Dynakube.Status.Synthetic.LastUpdateProbeTimestamp, ProbeThreshold)
}

type imageUpdater struct {
	now         metav1.Time
	dockerCfg   *dockerconfig.DockerConfig
	verProvider VersionProviderCallback
}

func (updater *imageUpdater) update(image toUpdateImage) error {
	image.version.LastUpdateProbeTimestamp = &updater.now

	ver, err := updater.verProvider(image.url, updater.dockerCfg)
	if err != nil {
		return errors.WithMessage(err, "failed to get image version")
	}

	if image.version.Version == ver.Version {
		return nil
	}

	if !image.allowsDowngrade && image.version.Version != "" {
		if upgrade, err := version.NeedsUpgradeRaw(image.version.Version, ver.Version); err != nil {
			return err
		} else if !upgrade {
			return nil
		}
	}

	log.Info("update found",
		"image", image.url,
		"oldVersion", image.version.Version,
		"newVersion", ver.Version,
		"oldHash", image.version.ImageHash,
		"newHash", ver.Hash)
	image.version.Version = ver.Version
	image.version.ImageHash = ver.Hash
	return nil
}
