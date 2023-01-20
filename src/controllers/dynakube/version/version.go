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

type updateScope struct {
	onActiveGate bool
	onEec        bool
	onStats      bool
	onOneAgent   bool
	onSynthetic  bool
	onDynaMetric bool
}

func (scope updateScope) needsUpdate() bool {
	return scope.onActiveGate ||
		scope.onEec ||
		scope.onStats ||
		scope.onOneAgent ||
		scope.onSynthetic ||
		scope.onDynaMetric
}

func (scope updateScope) reconcileActiveGate(dynakube *dynatracev1beta1.DynaKube, updater *imageUpdater) {
	if !scope.onActiveGate {
		return
	}

	err := updater.update(
		dynakube.ActiveGateImage(),
		&dynakube.Status.ActiveGate.VersionStatus,
		true)
	if err != nil {
		log.Error(err, "failed to update ActiveGate image version")
	}
}

func (scope updateScope) reconcileEec(dynakube *dynatracev1beta1.DynaKube, updater *imageUpdater) {
	if !scope.onEec {
		return
	}

	err := updater.update(
		dynakube.EecImage(),
		&dynakube.Status.ExtensionController.VersionStatus,
		true)
	if err != nil {
		log.Error(err, "Failed to update Extension Controller image version")
	}
}

func (scope updateScope) reconcileStatsd(dynakube *dynatracev1beta1.DynaKube, updater *imageUpdater) {
	if !scope.onStats {
		return
	}

	err := updater.update(
		dynakube.StatsdImage(),
		&dynakube.Status.Statsd.VersionStatus,
		true)
	if err != nil {
		log.Error(err, "Failed to update StatsD image version")
	}
}

func (scope updateScope) reconcileOneAgent(dynakube *dynatracev1beta1.DynaKube, updater *imageUpdater) {
	if !scope.onOneAgent {
		return
	}

	err := updater.update(
		dynakube.OneAgentImage(),
		&dynakube.Status.OneAgent.VersionStatus,
		false)
	if err != nil {
		log.Error(err, "failed to update OneAgent image version")
	}
}

func (scope updateScope) reconcileSynthetic(dynaKube *dynatracev1beta1.DynaKube, updater *imageUpdater) {
	if !scope.onSynthetic {
		return
	}

	err := updater.update(
		dynaKube.SyntheticImage(),
		&dynaKube.Status.Synthetic.VersionStatus,
		true)
	if err != nil {
		log.Error(err, "failed to update synthetic image version")
	}
}

func (scope updateScope) reconcileDynaMetrics(dynaKube *dynatracev1beta1.DynaKube, updater *imageUpdater) {
	if !scope.onDynaMetric {
		return
	}

	err := updater.update(
		dynaKube.DynaMetricImage(),
		&dynaKube.Status.DynaMetrics.VersionStatus,
		true)
	if err != nil {
		log.Error(err, "failed to update DynaMetric image version")
	}
}

// Reconcile updates the version and hash for the images used by the rec.Dynakube DynaKube instance.
func (r *Reconciler) Reconcile(ctx context.Context) error {
	scope := updateScope{
		onActiveGate: needsActiveGateUpdate(r.Dynakube, *r.TimeProvider),
		onEec:        needsEecUpdate(r.Dynakube, *r.TimeProvider),
		onStats:      needsStatsdUpdate(r.Dynakube, *r.TimeProvider),
		onOneAgent:   needsOneAgentUpdate(r.Dynakube, *r.TimeProvider),
		onSynthetic:  needsSynMonitoringUpdate(r.Dynakube, *r.TimeProvider),
		onDynaMetric: needsDynaMetricUpdate(r.Dynakube, *r.TimeProvider),
	}

	if !scope.needsUpdate() {
		return nil
	}

	return r.updateImages(ctx, scope)
}

func (r *Reconciler) updateImages(ctx context.Context, scope updateScope) error {
	dockerConfig, err := createDockerConfigWithCustomCAs(ctx, r.Dynakube, r.ApiReader, r.Fs)
	if err != nil {
		return err
	}

	imageUpdater := &imageUpdater{
		now:         *r.TimeProvider.Now(),
		dockerCfg:   dockerConfig,
		verProvider: r.VersionProvider,
	}
	scope.reconcileActiveGate(r.Dynakube, imageUpdater)
	scope.reconcileEec(r.Dynakube, imageUpdater)
	scope.reconcileStatsd(r.Dynakube, imageUpdater)
	scope.reconcileOneAgent(r.Dynakube, imageUpdater)
	scope.reconcileSynthetic(r.Dynakube, imageUpdater)
	scope.reconcileDynaMetrics(r.Dynakube, imageUpdater)

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

func needsSynMonitoringUpdate(dynakube *dynatracev1beta1.DynaKube, timeProvider kubeobjects.TimeProvider) bool {
	return dynakube.IsSyntheticMonitoringEnabled() &&
		dynakube.FeatureCustomSyntheticImage() == "" &&
		timeProvider.IsOutdated(dynakube.Status.Synthetic.LastUpdateProbeTimestamp, ProbeThreshold)
}

func needsDynaMetricUpdate(dynakube *dynatracev1beta1.DynaKube, timeProvider kubeobjects.TimeProvider) bool {
	return dynakube.IsSyntheticMonitoringEnabled() &&
		dynakube.FeatureCustomDynaMetricImage() == "" &&
		timeProvider.IsOutdated(dynakube.Status.DynaMetrics.LastUpdateProbeTimestamp, ProbeThreshold)
}

type imageUpdater struct {
	now         metav1.Time
	dockerCfg   *dockerconfig.DockerConfig
	verProvider VersionProviderCallback
}

func (updater *imageUpdater) update(
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
