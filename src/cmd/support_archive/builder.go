package support_archive

import (
	"context"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/spf13/cobra"
)

const (
	use               = "support-archive"
	namespaceFlagName = "namespace"
	stdoutFlagName    = "stdout"
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
		ctx := supportArchiveContext{
			ctx:           context.TODO(),
			namespaceName: namespaceFlagValue,
			toStdout:      stdoutFlagValue,
		}

		registerSupportArchiveLogger(&ctx)
		version.LogVersionToLogger(ctx.log)

		err := dynatracev1beta1.AddToScheme(scheme.Scheme)
		if err != nil {
			return err
		}

		supportArchive, err := newTarball(&ctx)
		if err != nil {
			return err
		}
		defer supportArchive.close()

		runCollectors(&ctx, supportArchive)
		printCopyCommand(&ctx, supportArchive)

		return nil
	}
}

func runCollectors(ctx *supportArchiveContext, supportArchive *tarball) {
	collectors := []func(*supportArchiveContext, *tarball) error{
		collectOperatorVersion,
	}

	for _, c := range collectors {
		if err := c(ctx, supportArchive); err != nil {
			logErrorf(ctx.log, err, "failed collector")
		}
	}
}

func printCopyCommand(ctx *supportArchiveContext, supportArchive *tarball) {
	if !ctx.toStdout {
		logInfof(ctx.log, "kubectl -n %s cp %s:%s .%s\n",
			os.Getenv("POD_NAMESPACE"),
			os.Getenv("POD_NAME"),
			supportArchive.tarFile.Name(),
			supportArchive.tarFile.Name())
	}
}
