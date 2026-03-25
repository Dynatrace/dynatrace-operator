package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	k8sversion "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

func checkKubernetesVersion(baseLog logd.Logger, kubeConfig *rest.Config) {
	log := baseLog.WithName("k8s")

	logNewCheckf(log, "checking Kubernetes version ...")

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		logWarningf(log, "could not create discovery client: %v", err)

		return
	}

	serverVersion, err := k8sversion.GetServerVersion(discoveryClient)
	if err != nil {
		logWarningf(log, "could not retrieve Kubernetes version: %v", err)

		return
	}

	logOkf(log, "%s (%s, %s)", serverVersion.GitVersion, serverVersion.Platform, serverVersion.GoVersion)
}
