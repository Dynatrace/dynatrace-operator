//go:build e2e

package operator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

const (
	helmRegistryURL = "oci://public.ecr.aws/dynatrace/dynatrace-operator"
)

// Install the operator chart with the specified tag and CSI mode.
func Install(releaseTag string, withCSI bool) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		if releaseTag == "" {
			return ctx, errors.New("missing release tag")
		}
		err := installViaHelm(releaseTag, withCSI)
		if err != nil {
			return ctx, err
		}

		return VerifyInstall(ctx, envConfig, withCSI)
	}
}

// InstallLocal deploys the operator helm chart from filesystem.
func InstallLocal(withCSI bool) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		err := installViaHelm("", withCSI)
		if err != nil {
			return ctx, err
		}

		return VerifyInstall(ctx, envConfig, withCSI)
	}
}

func Uninstall(withCSI bool) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		rootDir := project.RootDir()
		if withCSI {
			ctx, err := csi.CleanUpEachPod(DefaultNamespace)(ctx, envConfig)
			if err != nil {
				return ctx, err
			}
		}

		return ctx, execMakeCommand(rootDir, "undeploy", fmt.Sprintf("ENABLE_CSI=%t", withCSI))
	}
}

func VerifyInstall(ctx context.Context, envConfig *envconf.Config, withCSI bool) (context.Context, error) {
	ctx, err := WaitForDeployment(DefaultNamespace)(ctx, envConfig)
	if err != nil {
		return ctx, err
	}
	ctx, err = webhook.WaitForDeployment(DefaultNamespace)(ctx, envConfig)
	if err != nil {
		return ctx, err
	}

	if withCSI {
		ctx, err = csi.WaitForDaemonset(DefaultNamespace)(ctx, envConfig)
		if err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func execMakeCommand(rootDir, makeTarget string, envVariables ...string) error {
	command := exec.Command("make", "-C", rootDir, makeTarget)
	command.Env = os.Environ()
	command.Env = append(command.Env, envVariables...)
	b, bErr := new(bytes.Buffer), new(bytes.Buffer)
	command.Stdout = b
	command.Stderr = bErr
	err := command.Run()

	if len(b.String()) != 0 {
		fmt.Println("out:", b.String()) //nolint:forbidigo
	}

	if len(bErr.String()) != 0 {
		fmt.Println("err:", bErr.String()) //nolint:forbidigo
	}

	return err
}

func installViaHelm(releaseTag string, withCSI bool) error {
	manager := helm.New("''")

	_platform, err := platform.NewResolver().GetPlatform()
	if err != nil {
		return err
	}

	opts := []helm.Option{
		helm.WithReleaseName("dynatrace-operator"),
		helm.WithNamespace("dynatrace"),
		helm.WithArgs("--create-namespace"),
		helm.WithArgs("--install"),
		helm.WithArgs("--set", fmt.Sprintf("platform=%s", _platform)),
		helm.WithArgs("--set", "installCRD=true"),
		helm.WithArgs("--set", fmt.Sprintf("csidriver.enabled=%t", withCSI)),
		helm.WithArgs("--set", "manifests=true"),
		helm.WithArgs("--set", "debugLogs=true"),
	}

	if releaseTag == "" {
		// Install from filesystem
		rootDir := project.RootDir()
		imageRef, err := getImageRef(rootDir)
		if err != nil {
			return err
		}

		if imageRef == "" {
			return errors.New("could not determine operator image")
		}

		opts = append(opts,
			helm.WithArgs(filepath.Join(rootDir, "config", "helm", "chart", "default")),
			helm.WithArgs("--set", "image="+strings.TrimSpace(imageRef)),
		)
	} else {
		// Install from registry
		opts = append(opts,
			helm.WithArgs(helmRegistryURL),
			helm.WithVersion(releaseTag),
		)
	}

	var klogLevel klog.Level
	// Show helm command args and output
	// Set only fails if the input does not conform to stronv.ParseInt(x, 10, 32)
	_ = klogLevel.Set("4")
	defer func() {
		// Reset to default value to prevent other logs from showing up
		_ = klogLevel.Set("0")
	}()

	return manager.RunUpgrade(opts...)
}

func getImageRef(rootDir string) (string, error) {
	command := exec.Command("make", "-C", rootDir, "deploy/show-image-ref")
	command.Env = os.Environ()
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	command.Stdout = stdout
	command.Stderr = stderr
	err := command.Run()
	if err != nil || stderr.String() != "" {
		return "", fmt.Errorf("%s: %w", stderr, err)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	stdout.Reset()
	for _, line := range lines {
		// make prints things to stdout, e.g. make[1]: Entering directory
		if !strings.HasPrefix(line, "make[") {
			stdout.WriteString(line)
		}
	}

	return stdout.String(), nil
}
