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

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		err := InstallViaHelm(releaseTag, withCSI,
			helm.WithArgs("--create-namespace"),
			helm.WithArgs("--install"),
		)
		if err != nil {
			return ctx, err
		}

		return VerifyInstall(ctx, envConfig, withCSI)
	}
}

// InstallLocal deploys the operator helm chart from filesystem.
func InstallLocal(withCSI bool) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		if os.Getenv("OLM") == "true" {
			if withCSI {
				fmt.Println("skipping CSI tests with OLM installation") //nolint:forbidigo
				envConfig.WithSkipFeatureRegex(".*")

				return ctx, nil
			}
			err := installViaOLMLocalBundle()
			if err != nil {
				return ctx, err
			}
		} else {
			err := InstallViaHelm("", withCSI,
				helm.WithArgs("--create-namespace"),
				helm.WithArgs("--install"),
			)
			if err != nil {
				return ctx, err
			}
		}

		return VerifyInstall(ctx, envConfig, withCSI)
	}
}

func InstallWithHelmAsUser(releaseTag string, withCSI bool, serviceAccount string) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		err := InstallViaHelm(releaseTag, withCSI,
			helm.WithArgs("--create-namespace"),
			helm.WithArgs("--install"),
			helm.WithArgs("--kube-as-user", serviceAccount),
		)
		if err != nil {
			return ctx, err
		}

		return VerifyInstall(ctx, envConfig, withCSI)
	}
}

func installViaOLMLocalBundle() error {
	err := execMakeCommand(project.RootDir(), "bundle/show-image-ref")
	if err != nil {
		return err
	}

	return execMakeCommand(project.RootDir(), "bundle/run")
}

func Uninstall(withCSI bool) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		rootDir := project.RootDir()

		if os.Getenv("OLM") == "true" {
			return ctx, execMakeCommand(rootDir, "bundle/cleanup")
		} else {
			if withCSI {
				ctx, err := csi.CleanUpEachPod(DefaultNamespace)(ctx, envConfig)
				if err != nil {
					return ctx, err
				}
			}

			return ctx, execMakeCommand(rootDir, "undeploy", fmt.Sprintf("ENABLE_CSI=%t", withCSI))
		}
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

	ctx, err = PrintDeploymentMetadata(ctx, envConfig)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}

func PrintDeploymentMetadata(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
	resource := envConfig.Client().Resources()

	return ctx, k8sdeployment.NewQuery(ctx, resource, client.ObjectKey{
		Name:      DeploymentName,
		Namespace: DefaultNamespace,
	}).ForEachPod(func(pod corev1.Pod) {
		fmt.Printf("Metadata for all containers for %s\n", DeploymentName) //nolint:forbidigo
		for _, container := range pod.Status.ContainerStatuses {
			fmt.Printf("\tcontainer name: %s\n", container.Name) //nolint:forbidigo
			fmt.Printf("\timage: %s\n", container.Image)         //nolint:forbidigo
			fmt.Printf("\timageID: %s\n", container.ImageID)     //nolint:forbidigo
		}
	})
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

// RunHelmUpgrade resolves the base helm options for a local chart and runs helm upgrade
// with any additional caller-supplied options appended. It sets klog verbosity so helm
// command args and output are visible in test logs.
// Use this for both installs (pass --install/--create-namespace as extraOpts) and plain upgrades.
func RunHelmUpgrade(withCSI bool, extraOpts ...helm.Option) error {
	opts, err := GetHelmBaseOptions(withCSI)
	if err != nil {
		return err
	}

	var klogLevel klog.Level
	// Show helm command args and output
	// Set only fails if the input does not conform to stronv.ParseInt(x, 10, 32)
	_ = klogLevel.Set("4")
	defer func() {
		// Reset to default value to prevent other logs from showing up
		_ = klogLevel.Set("0")
	}()

	return helm.New("''").RunUpgrade(append(opts, extraOpts...)...)
}

// InstallViaHelm runs helm upgrade for the operator chart. It resolves the chart source and
// platform, then appends extraOpts — so callers control whether this is a fresh install
// (pass helm.WithArgs("--install"), helm.WithArgs("--create-namespace")) or a plain upgrade.
func InstallViaHelm(releaseTag string, withCSI bool, extraOpts ...helm.Option) error {
	// Registry installs cannot use GetHelmBaseOptions — build opts directly.
	if releaseTag != "" {
		_platform, err := platform.NewResolver().GetPlatform()
		if err != nil {
			return err
		}

		opts, err := getHelmOptions(releaseTag, _platform, withCSI)
		if err != nil {
			return err
		}

		var klogLevel klog.Level
		_ = klogLevel.Set("4")
		defer func() { _ = klogLevel.Set("0") }()

		return helm.New("''").RunUpgrade(append(opts, extraOpts...)...)
	}

	_platform, err := platform.NewResolver().GetPlatform()
	if err != nil {
		return err
	}

	return RunHelmUpgrade(withCSI,
		append([]helm.Option{helm.WithArgs("--set", fmt.Sprintf("platform=%s", _platform))}, extraOpts...)...,
	)
}

// GetHelmBaseOptions returns helm options common to both install and upgrade for a local chart.
// It handles image resolution and chart source selection (TARGET_BRANCH, HELM_CHART env vars)
// but does not include --install or --create-namespace so callers can use it for either operation.
func GetHelmBaseOptions(withCSI bool) ([]helm.Option, error) {
	opts := []helm.Option{
		helm.WithReleaseName("dynatrace-operator"),
		helm.WithNamespace("dynatrace"),
		helm.WithArgs("--set", "installCRD=true"),
		helm.WithArgs("--set", fmt.Sprintf("csidriver.enabled=%t", withCSI)),
		helm.WithArgs("--set", "manifests=true"),
		helm.WithArgs("--set", "debugLogs=true"),
	}

	rootDir := project.RootDir()
	isFIPS := os.Getenv("FIPS") == "true"
	imageRef, err := GetImageRef(rootDir, isFIPS)
	if err != nil {
		return nil, err
	}
	if imageRef == "" {
		return nil, errors.New("could not determine operator image")
	}

	// if target branch is set and not main, it means that we are running tests on a feature branch, so we want to
	// install the operator using local helm chart and target branch
	if targetBranch, ok := os.LookupEnv("TARGET_BRANCH"); ok && targetBranch != "main" {
		return append(opts,
			helm.WithArgs("--set", "image="+strings.TrimSpace(imageRef)),
			helm.WithArgs("--set", "imageRef.pullPolicy=Always"),
			helm.WithArgs(filepath.Join(rootDir, "config", "helm", "chart", "default")),
		), nil
	}

	// Install nightly
	if chartURI := os.Getenv("HELM_CHART"); strings.HasSuffix(chartURI, ":0.0.0-nightly-chart") {
		return append(opts, helm.WithArgs(chartURI)), nil
	}

	return append(opts,
		helm.WithArgs("--set", "image="+strings.TrimSpace(imageRef)),
		helm.WithArgs("--set", "imageRef.pullPolicy=Always"),
		helm.WithArgs(filepath.Join(rootDir, "config", "helm", "chart", "default")),
	), nil
}

func getHelmOptions(releaseTag, platform string, withCSI bool) ([]helm.Option, error) {
	// Install from registry — build options directly, no local chart involved.
	if releaseTag != "" {
		return []helm.Option{
			helm.WithReleaseName("dynatrace-operator"),
			helm.WithNamespace("dynatrace"),
			helm.WithArgs("--create-namespace"),
			helm.WithArgs("--install"),
			helm.WithArgs("--set", fmt.Sprintf("platform=%s", platform)),
			helm.WithArgs("--set", "installCRD=true"),
			helm.WithArgs("--set", fmt.Sprintf("csidriver.enabled=%t", withCSI)),
			helm.WithArgs("--set", "manifests=true"),
			helm.WithArgs("--set", "debugLogs=true"),
			helm.WithArgs(helmRegistryURL),
			helm.WithVersion(releaseTag),
		}, nil
	}

	base, err := GetHelmBaseOptions(withCSI)
	if err != nil {
		return nil, err
	}

	return append([]helm.Option{
		helm.WithArgs("--create-namespace"),
		helm.WithArgs("--install"),
		helm.WithArgs("--set", fmt.Sprintf("platform=%s", platform)),
	}, base...), nil
}

// Cache image ref on first invocation to allow switching branches.
var imageRef string

func GetImageRef(rootDir string, fips140 bool) (string, error) {
	if imageRef == "" {
		cmdShowImage := "deploy/show-image-ref"

		if fips140 {
			cmdShowImage = "deploy/show-image-ref/fips"
		}

		command := exec.Command("make", "-C", rootDir, cmdShowImage)
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
			if !strings.HasPrefix(line, "make") {
				stdout.WriteString(line)
			}
		}

		imageRef = stdout.String()
	}

	return imageRef, nil
}
