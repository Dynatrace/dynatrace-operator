package operator

import (
	"net/url"
	"path"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	localManifestsDir          = "config/deploy/kubernetes/"
	kubernetesCsiManifest      = "kubernetes-csi.yaml"
	kubernetesOperatorManifest = "kubernetes.yaml"
	openshiftCsiManifest       = "openshift-csi.yaml"
	openshiftOperatorManifest  = "openshift.yaml"
)

func InstallFromSource(withCsi bool) features.Func {
	paths := manifestsPaths(withCsi)
	return manifests.InstallFromFiles(paths)
}

func manifestsPaths(withCsi bool) []string {
	platform := kubeobjects.ResolvePlatformFromEnv()
	paths := []string{}

	switch platform {
	case kubeobjects.Openshift:
		paths = append(paths, path.Join(project.RootDir(), localManifestsDir, kubernetesOperatorManifest))
		if withCsi {
			paths = append(paths, path.Join(project.RootDir(), localManifestsDir, kubernetesCsiManifest))
		}
	default:
		paths = append(paths, path.Join(project.RootDir(), localManifestsDir, kubernetesOperatorManifest))
		if withCsi {
			paths = append(paths, path.Join(project.RootDir(), localManifestsDir, kubernetesCsiManifest))
		}
	}

	return paths
}

func InstallFromGithub(releaseTag string, withCsi bool) features.Func {
	manifestsUrls := manifestsUrls(releaseTag, withCsi)
	return manifests.InstallFromUrls(manifestsUrls)
}

func manifestsUrls(releaseTag string, withCsi bool) []string {
	const dynatraceOperatorGithubDownloadUrl = "https://github.com/Dynatrace/dynatrace-operator/releases/download/"
	platform := kubeobjects.ResolvePlatformFromEnv()

	manifestsUrls := []string{}
	switch platform {
	case kubeobjects.Openshift:
		openshiftOperatorManifestsUrl, _ := url.JoinPath(dynatraceOperatorGithubDownloadUrl, releaseTag, openshiftOperatorManifest)
		openshiftCsiManifestsUrl, _ := url.JoinPath(dynatraceOperatorGithubDownloadUrl, releaseTag, openshiftCsiManifest)
		manifestsUrls = append(manifestsUrls, openshiftOperatorManifestsUrl)
		if withCsi {
			manifestsUrls = append(manifestsUrls, openshiftCsiManifestsUrl)
		}
	default:
		kubernetesOperatorManifestsUrl, _ := url.JoinPath(dynatraceOperatorGithubDownloadUrl, releaseTag, kubernetesOperatorManifest)
		kubernetesCsiManifestsUrl, _ := url.JoinPath(dynatraceOperatorGithubDownloadUrl, releaseTag, kubernetesCsiManifest)
		manifestsUrls = append(manifestsUrls, kubernetesOperatorManifestsUrl)
		if withCsi {
			manifestsUrls = append(manifestsUrls, kubernetesCsiManifestsUrl)
		}
	}
	return manifestsUrls
}

func WaitForDeployment() features.Func {
	return deployment.WaitFor("dynatrace-operator", "dynatrace")
}
