package operator

import (
	"os"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	platformEnv       = "PLATFORM"
	openshiftPlatform = "openshift"
)

func InstallAll() features.Func {
	platform := os.Getenv(platformEnv)
	if platform == openshiftPlatform {
		return InstallAllForOpenshift()
	} else {
		return InstallAllForKubernetes()
	}
}

func Install() features.Func {
	platform := os.Getenv(platformEnv)
	if platform == openshiftPlatform {
		return InstallForOpenshift()
	} else {
		return InstallForKubernetes()
	}
}

func InstallAllForKubernetes() features.Func {
	return manifests.InstallFromFile("../../config/deploy/kubernetes/kubernetes-all.yaml")
}

func InstallForKubernetes() features.Func {
	return manifests.InstallFromFile("../../config/deploy/kubernetes/kubernetes.yaml")
}

func InstallAllForOpenshift() features.Func {
	return manifests.InstallFromFile("../../config/deploy/openshift/openshift-all.yaml")
}

func InstallForOpenshift() features.Func {
	return manifests.InstallFromFile("../../config/deploy/openshift/openshift.yaml")
}

const (
	manifestsWithCsi    = "../../config/deploy/kubernetes/kubernetes-all.yaml"
	manifestsWithoutCsi = "../../config/deploy/kubernetes/kubernetes.yaml"
)

func InstallOperatorFromSource(withCsi bool) features.Func {
	actualManifestPath := manifestsWithoutCsi
	if withCsi {
		actualManifestPath = manifestsWithCsi
	}

	return manifests.InstallFromLocalFile(actualManifestPath)
}

func InstallOperatorFromGithub(releaseTag string, withCsi bool) features.Func {
	manifestsUrl := fmt.Sprintf("https://github.com/Dynatrace/dynatrace-operator/releases/download/%s/", releaseTag)
	if withCsi {
		manifestsUrl += "/kubernetes.yaml"
	} else {
		manifestsUrl += "/kubernetes-all.yaml"
	}

	return manifests.InstallFromUrl(manifestsUrl)
}

func WaitForDeployment() features.Func {
	return deployment.WaitFor("dynatrace-operator", "dynatrace")
}
