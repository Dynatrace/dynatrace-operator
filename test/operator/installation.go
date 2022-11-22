package operator

import (
	"os"
	"net/url"
	"path"
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
	localManifestsDir = "../../config/deploy/kubernetes/"
	csiManifest       = "kubernetes-csi.yaml"
	operatorManifest  = "kubernetes.yaml"
)

func InstallOperatorFromSource(withCsi bool) features.Func {
	paths := []string{path.Join(localManifestsDir, operatorManifest)}

	if withCsi {
		paths = append(paths, path.Join(localManifestsDir, csiManifest))
	}

	return manifests.InstallFromFiles(paths)
}

func InstallOperatorFromGithub(releaseTag string, withCsi bool) features.Func {
	const dynatraceOperatorGithubDownloadUrl = "https://github.com/Dynatrace/dynatrace-operator/releases/download/"

	operatorManifestsUrl, _ := url.JoinPath(dynatraceOperatorGithubDownloadUrl, releaseTag, operatorManifest)
	csiManifestsUrl, _ := url.JoinPath(dynatraceOperatorGithubDownloadUrl, releaseTag, csiManifest)

	manifestsUrls := []string{operatorManifestsUrl}
	if withCsi {
		manifestsUrls = append(manifestsUrls, csiManifestsUrl)
	}

	return manifests.InstallFromUrls(manifestsUrls)
}

func WaitForDeployment() features.Func {
	return deployment.WaitFor("dynatrace-operator", "dynatrace")
}
