//go:build e2e

package operator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

const (
	helmRegistryUrl = "oci://public.ecr.aws/dynatrace/dynatrace-operator"
)

func InstallViaMake(withCSI bool) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		rootDir := project.RootDir()
		err := execMakeCommand(rootDir, "deploy/helm", fmt.Sprintf("ENABLE_CSI=%t", withCSI))
		if err != nil {
			return ctx, err
		}
		ctx, err = VerifyInstall(ctx, envConfig)

		return ctx, err
	}
}

func InstallViaHelm(releaseTag string, withCsi bool, namespace string) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		err := installViaHelm(releaseTag, withCsi, namespace)
		if err != nil {
			return ctx, err
		}

		return VerifyInstall(ctx, envConfig)
	}
}

func UninstallViaMake(withCSI bool) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		rootDir := project.RootDir()
		if withCSI {
			ctx, err := csi.CleanUpEachPod(DefaultNamespace)(ctx, envConfig)
			if err != nil {
				return ctx, err
			}
		}

		return ctx, execMakeCommand(rootDir, "undeploy/helm", fmt.Sprintf("ENABLE_CSI=%t", withCSI))
	}
}

func VerifyInstall(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
	ctx, err := WaitForDeployment(DefaultNamespace)(ctx, envConfig)
	if err != nil {
		return ctx, err
	}
	ctx, err = webhook.WaitForDeployment(DefaultNamespace)(ctx, envConfig)
	if err != nil {
		return ctx, err
	}
	ctx, err = csi.WaitForDaemonset(DefaultNamespace)(ctx, envConfig)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}

func execMakeCommand(rootDir, makeTarget string, envVariables ...string) error {
	command := exec.Command("make", "-C", rootDir, makeTarget)
	command.Env = os.Environ()
	command.Env = append(command.Env, envVariables...)
	b := new(bytes.Buffer)
	command.Stdout = b
	err := command.Run()
	fmt.Println("out:", b.String()) //nolint

	return err
}

func installViaHelm(releaseTag string, withCsi bool, namespace string) error {
	manager := helm.New("''")

	_platform, err := platform.NewResolver().GetPlatform()
	if err != nil {
		return err
	}

	return manager.RunUpgrade(helm.WithNamespace(namespace),
		helm.WithReleaseName("dynatrace-operator"),
		helm.WithArgs(helmRegistryUrl),
		helm.WithVersion(releaseTag),
		helm.WithArgs("--create-namespace"),
		helm.WithArgs("--install"),
		helm.WithArgs("--set", fmt.Sprintf("platform=%s", _platform)),
		helm.WithArgs("--set", "installCRD=true"),
		helm.WithArgs("--set", fmt.Sprintf("csidriver.enabled=%t", withCsi)),
		helm.WithArgs("--set", "manifests=true"),
	)
}
