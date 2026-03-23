package supportarchive

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"k8s.io/client-go/discovery"
)

const kubernetesVersionCollectorName = "kubernetesVersionCollector"

type kubernetesVersionCollector struct {
	collectorCommon
	discoveryClient discovery.DiscoveryInterface
}

func newKubernetesVersionCollector(log logd.Logger, supportArchive archiver, discoveryClient discovery.DiscoveryInterface) collector {
	return kubernetesVersionCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		discoveryClient: discoveryClient,
	}
}

func (kvc kubernetesVersionCollector) Do() error {
	logInfof(kvc.log, "Storing Kubernetes version into %s", KubernetesVersionFileName)

	serverVersion, err := kvc.discoveryClient.ServerVersion()
	if err != nil {
		logErrorf(kvc.log, err, "Failed to retrieve Kubernetes server version")
		return err
	}

	versionString := fmt.Sprintf(
		"major: %s\nminor: %s\ngitVersion: %s\ngitCommit: %s\ngitTreeState: %s\nbuildDate: %s\nplatform: %s\n",
		serverVersion.Major,
		serverVersion.Minor,
		serverVersion.GitVersion,
		serverVersion.GitCommit,
		serverVersion.GitTreeState,
		serverVersion.BuildDate,
		serverVersion.Platform,
	)

	if err := kvc.supportArchive.addFile(KubernetesVersionFileName, strings.NewReader(versionString)); err != nil {
		return err
	}

	return nil
}

func (kvc kubernetesVersionCollector) Name() string {
	return kubernetesVersionCollectorName
}
