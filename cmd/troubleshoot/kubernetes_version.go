package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	k8sversion "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

func checkKubernetesVersion(baseLog logd.Logger, kubeConfig *rest.Config) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		logWarningf(baseLog, "could not create Kubernetes discovery client: %v", err)

		return
	}

	serverVersion, err := k8sversion.GetServerVersion(discoveryClient)
	if err != nil {
		logWarningf(baseLog, "could not retrieve Kubernetes version: %v", err)

		return
	}

	logInfof(baseLog, "Kubernetes: %s (%s, %s)", serverVersion.GitVersion, serverVersion.Platform, serverVersion.GoVersion)
}
