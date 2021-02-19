package utils

import (
	"fmt"
	"strings"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const updateInterval = 5 * time.Minute

// SetUseImmutableImageStatus sets the UseImmutableImage and LastClusterVersionProbeTimestamp stati of an BaseOneAgentDaemonSet instance
// Returns true if:
//     UseImmutableImage of specification is true &&
//			LastClusterVersionProbeTimestamp status is the duration of updateInterval behind
// otherwise returns false
func SetUseImmutableImageStatus(logger logr.Logger, instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, dtc dtclient.Client) bool {
	if dtc == nil {
		err := fmt.Errorf("dynatrace client is nil")
		logger.Error(err, err.Error())
		return false
	}

	if !fs.UseImmutableImage {
		return false
	}
	if ts := instance.Status.LastClusterVersionProbeTimestamp; ts != nil && !isLastProbeOutdated(ts.UTC()) {
		return false
	}

	now := metav1.Now()
	instance.Status.LastClusterVersionProbeTimestamp = &now

	logger.Info("Getting agent version")
	agentVersion, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		logger.Error(err, err.Error())
		return true
	}

	logger.Info("Getting cluster version")
	clusterInfo, err := dtc.GetClusterInfo()
	if err != nil {
		logger.Error(err, err.Error())
		return true
	}

	if clusterInfo == nil {
		err = fmt.Errorf("could not retrieve cluster info")
		logger.Error(err, err.Error())
		return true
	}

	logger.Info("Comparing versions with minimum versions", "clusterVersion", clusterInfo.Version, "agentVersion", agentVersion)
	instance.Status.OneAgent.UseImmutableImage =
		version.IsRemoteClusterVersionSupported(logger, clusterInfo.Version) &&
			version.IsAgentVersionSupported(logger, agentVersion)

	return true
}

func isLastProbeOutdated(ts time.Time) bool {
	return metav1.Now().UTC().Sub(ts) > updateInterval
}

func BuildActiveGateImage(instance *dynatracev1alpha1.DynaKube) string {
	if instance.Spec.ActiveGate.Image != "" {
		return instance.Spec.ActiveGate.Image
	}
	return buildActiveGateImage(instance)
}

func BuildPullSecret(instance *dynatracev1alpha1.DynaKube) corev1.LocalObjectReference {
	return corev1.LocalObjectReference{
		Name: buildPullSecretName(instance),
	}
}

func BuildOneAgentImage(instance *dynatracev1alpha1.DynaKube, agentVersion string) (string, error) {
	registry := buildImageRegistryFromAPIURL(instance.Spec.APIURL)
	image := registry + "/linux/oneagent"

	if agentVersion != "" {
		image += ":" + agentVersion
	}

	return image, nil
}

func buildPullSecretName(instance *dynatracev1alpha1.DynaKube) string {
	name := instance.Name + dtpullsecret.PullSecretSuffix
	if instance.Spec.CustomPullSecret != "" {
		name = instance.Spec.CustomPullSecret
	}
	return name
}

func buildActiveGateImage(instance *dynatracev1alpha1.DynaKube) string {
	registry := buildImageRegistryFromAPIURL(instance.Spec.APIURL)
	return fmt.Sprintf("%s/linux/activegate", registry)
}

func buildImageRegistryFromAPIURL(apiURL string) string {
	r := strings.TrimPrefix(apiURL, "https://")
	r = strings.TrimPrefix(r, "http://")
	r = strings.TrimSuffix(r, "/api")
	return r
}
