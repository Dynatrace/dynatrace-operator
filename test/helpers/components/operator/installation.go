//go:build e2e

package operator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

const (
	helmRepoUrl = "https://raw.githubusercontent.com/Dynatrace/dynatrace-operator/main/config/helm/repos/stable"
)

func InstallViaMake(withCSI bool) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		rootDir := project.RootDir()
		execMakeCommand(t, rootDir, "install", fmt.Sprintf("ENABLE_CSI=%t", withCSI))
		return ctx
	}
}

func InstallViaHelm(releaseTag string, withCsi bool, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		installViaHelm(t, releaseTag, withCsi, namespace)
		return ctx
	}
}

func UninstallViaMake(withCSI bool) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		rootDir := project.RootDir()
		err := execMakeCommand(t, rootDir, "undeploy/helm", fmt.Sprintf("ENABLE_CSI=%t", withCSI))
		if err != nil {
			t.Fatal("failed to execute make command to undeploy the operator", err)
		}
		return ctx
	}
}

func execMakeCommand(t *testing.T, rootDir, makeTarget string, envVariables ...string) error {
	command := exec.Command("make", "-C", rootDir, makeTarget)
	command.Env = os.Environ()
	command.Env = append(command.Env, envVariables...)

	output, err := command.CombinedOutput()
	if err != nil {
		t.Log(string(output))
	}

	return err
}

func installViaHelm(t *testing.T, releaseTag string, withCsi bool, namespace string) {
	manager := helm.New("''")
	err := manager.RunRepo(helm.WithArgs("add", "dynatrace", helmRepoUrl))
	if err != nil {
		t.Log("failed to add dynatrace helm chart repo", err)
	}

	err = manager.RunRepo(helm.WithArgs("install"))
	if err != nil {
		t.Fatal("failed to install helm repo")
	}

	err = manager.RunUpgrade(helm.WithName("dynatrace-operator"), helm.WithNamespace(namespace),
		helm.WithReleaseName("dynatrace/dynatrace-operator"),
		helm.WithVersion(releaseTag),
		helm.WithArgs("--create-namespace"),
		helm.WithArgs("--install"),
		helm.WithArgs("--set", fmt.Sprintf("platform=%s", platform.NewResolver().GetPlatform(t))),
		helm.WithArgs("--set", "installCRD=true"),
		helm.WithArgs("--set", fmt.Sprintf("csidriver.enabled=%t", withCsi)),
		helm.WithArgs("--set", "manifests=true"),
	)
	if err != nil {
		t.Fatal("failed to install dynatrace operator via helm")
	}
}
