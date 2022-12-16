package support_archive

import (
	"context"
	"io"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

const (
	use                            = "support-archive"
	namespaceFlagName              = "namespace"
	tarballToStdoutFlagName        = "stdout"
	defaultSupportArchiveTargetDir = "/tmp/dynatrace-operator"
)

var (
	namespaceFlagValue       string
	tarballToStdoutFlagValue bool
)

type CommandBuilder struct {
	configProvider config.Provider
}

func NewCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) SetConfigProvider(provider config.Provider) CommandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder CommandBuilder) Build() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		Long: "Pack logs and manifests useful for troubleshooting into single tarball",
		RunE: builder.buildRun(),
	}
	addFlags(cmd)
	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&namespaceFlagValue, namespaceFlagName, "dynatrace", "Specify a different Namespace.")
	cmd.PersistentFlags().BoolVar(&tarballToStdoutFlagValue, tarballToStdoutFlagName, false, "Write tarball to stdout.")
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log := newSupportArchiveLogger(getLogOutput(tarballToStdoutFlagValue))
		version.LogVersionToLogger(log)

		err := dynatracev1beta1.AddToScheme(scheme.Scheme)
		if err != nil {
			return errors.WithStack(err)
		}

		tarFile, err := createTarballTargetFile(tarballToStdoutFlagValue, defaultSupportArchiveTargetDir)
		if err != nil {
			return err
		}
		supportArchive := newTarball(tarFile)
		defer tarFile.Close()
		defer supportArchive.close()

		err = runCollectors(log, supportArchive, builder.configProvider)
		if err != nil {
			return err
		}
		printCopyCommand(log, tarballToStdoutFlagValue, tarFile.Name())

		return nil
	}
}

func getLogOutput(tarballToStdout bool) io.Writer {
	if tarballToStdout {
		// avoid corrupting tarball
		return os.Stderr
	} else {
		return os.Stdout
	}
}

func runCollectors(log logr.Logger, supportArchive tarball, configProvider config.Provider) error {
	context := context.Background()

	kubeConfig, err := configProvider.GetConfig()
	if err != nil {
		return err
	}

	clientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	collectors := []collector{
		newOperatorVersionCollector(log, supportArchive),
		newLogCollector(context, log, supportArchive, clientSet.CoreV1().Pods(namespaceFlagValue)),
	}

	for _, c := range collectors {
		if err := c.Do(); err != nil {
			logErrorf(log, err, "%s failed", c.Name())
		}
	}
	return nil
}

func printCopyCommand(log logr.Logger, tarballToStdout bool, tarFileName string) {
	podNamespace := os.Getenv("POD_NAMESPACE")
	podName := os.Getenv("POD_NAME")

	if tarballToStdout {
		return
	}

	if podNamespace == "" || podName == "" {
		// most probably not running on a pod
		logInfof(log, "cp %s .", tarFileName)
	} else {
		logInfof(log, "kubectl -n %s cp %s:%s .%s\n",
			podNamespace, podName, tarFileName, tarFileName)
	}
}
