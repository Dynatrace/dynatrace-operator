package operator

import (
	"context"
	"net/url"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	localKubernetesManifestsDir = "config/deploy/kubernetes/"
	localOpenshiftManifestsDir  = "config/deploy/openshift/"
	kubernetesCsiManifest       = "kubernetes-csi.yaml"
	kubernetesOperatorManifest  = "kubernetes.yaml"
	openshiftCsiManifest        = "openshift-csi.yaml"
	openshiftOperatorManifest   = "openshift.yaml"
)

func InstallFromSource(withCsi bool) features.Func {
	paths := manifestsPaths(withCsi)
	return manifests.InstallFromFiles(paths)
}

func InstallViaMake(withCSI bool) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		rootDir := project.RootDir()

		runBuildAndManifests(rootDir, t)

		platform := kubeobjects.ResolvePlatformFromEnv()
		makeTarget := getDeployMakeTarget(platform, withCSI, t)

		err := exec.Command("make", "-C", rootDir, makeTarget).Run()
		if err != nil {
			t.Fatal("failed to install the operator via the make command", err)
			return nil
		}

		return ctx
	}
}

func getDeployMakeTarget(platform kubeobjects.Platform, withCSI bool, t *testing.T) string {
	makeTarget := "deploy"
	switch platform {
	case kubeobjects.Openshift:
		makeTarget = strings.Join([]string{makeTarget, "openshift"}, "/")
	case kubeobjects.Kubernetes:
		makeTarget = strings.Join([]string{makeTarget, "kubernetes"}, "/")
	default:
		t.Fatal("failed to install the operator via the make command as no correct platform was set")
		return ""
	}

	if !withCSI {
		makeTarget = strings.Join([]string{makeTarget, "no-csi"}, "-")
	}

	return makeTarget
}

func runBuildAndManifests(rootDir string, t *testing.T) {
	err := exec.Command("make", "-C", rootDir, "build").Run()
	if err != nil {
		t.Fatal("failed to install the operator via the make command", err)
		return
	}

	err = exec.Command("make", "-C", rootDir, "manifests/branch").Run()
	if err != nil {
		t.Fatal("failed to install the operator via the make command", err)
		return
	}
}

func manifestsPaths(withCsi bool) []string {
	platform := kubeobjects.ResolvePlatformFromEnv()
	paths := []string{}

	switch platform {
	case kubeobjects.Openshift:
		paths = append(paths, path.Join(project.RootDir(), localOpenshiftManifestsDir, openshiftOperatorManifest))
		if withCsi {
			paths = append(paths, path.Join(project.RootDir(), localOpenshiftManifestsDir, openshiftCsiManifest))
		}
	default:
		paths = append(paths, path.Join(project.RootDir(), localKubernetesManifestsDir, kubernetesOperatorManifest))
		if withCsi {
			paths = append(paths, path.Join(project.RootDir(), localKubernetesManifestsDir, kubernetesCsiManifest))
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
