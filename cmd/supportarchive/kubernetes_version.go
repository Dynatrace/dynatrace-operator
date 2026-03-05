package supportarchive

import (
	"strings"

	k8sversion "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	"github.com/go-logr/logr"
	"k8s.io/client-go/discovery"
)

const kubernetesVersionCollectorName = "kubernetesVersionCollector"

type kubernetesVersionCollector struct {
	collectorCommon
	discoveryClient discovery.DiscoveryInterface
}

func newKubernetesVersionCollector(log logr.Logger, supportArchive archiver, discoveryClient discovery.DiscoveryInterface) collector {
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

	serverVersion, err := k8sversion.GetFormattedServerVersion(kvc.discoveryClient)
	if err != nil {
		logErrorf(kvc.log, err, "Failed to retrieve Kubernetes server version")

		return err
	}

	if err := kvc.supportArchive.addFile(KubernetesVersionFileName, strings.NewReader(serverVersion)); err != nil {
		return err
	}

	return nil
}

func (kvc kubernetesVersionCollector) Name() string {
	return kubernetesVersionCollectorName
}
