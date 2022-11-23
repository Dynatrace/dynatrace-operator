package support_archive

import (
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

const (
	use                            = "support-archive"
	namespaceFlagName              = "namespace"
	stdoutFlagName                 = "stdout"
	defaultSupportArchiveTargetDir = "/tmp/dynatrace-operator"
)

var (
	namespaceFlagValue string
	stdoutFlagValue    bool
)

type CommandBuilder struct {
}

func NewCommandBuilder() CommandBuilder {
	return CommandBuilder{}
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
	cmd.PersistentFlags().BoolVar(&stdoutFlagValue, stdoutFlagName, false, "Write tarball to stdout.")
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log := newSupportArchiveLogger(stdoutFlagValue)
		version.LogVersionToLogger(log)

		err := dynatracev1beta1.AddToScheme(scheme.Scheme)
		if err != nil {
			return err
		}

		tarFile, err := createTarballTargetFile(stdoutFlagValue, defaultSupportArchiveTargetDir)
		if err != nil {
			return err
		}
		supportArchive := newTarball(tarFile)
		defer tarFile.Close()
		defer supportArchive.close()

		runCollectors(log, supportArchive)
		printCopyCommand(log, stdoutFlagValue, tarFile.Name())

		return nil
	}
}

func runCollectors(log logr.Logger, supportArchive tarball) {
	collectors := []collector{
		operatorVersionCollector{
			collectorCommon{
				log:            log,
				supportArchive: supportArchive,
			},
		},
	}

	for _, c := range collectors {
		if err := c.Do(); err != nil {
			logErrorf(log, err, "%s failed", c.Name())
		}
	}
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
