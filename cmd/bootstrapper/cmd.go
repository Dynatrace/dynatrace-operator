package bootstrapper

import (
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper/download"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	use = "bootstrap"

	TargetFolderFlag   = cmd.TargetFolderFlag
	TargetVersionFlag  = "version"
	SuppressErrorsFlag = cmd.SuppressErrorsFlag
	TechnologiesFlag   = "technologies"
	FlavorFlag         = "flavor"
)

var (
	targetFolder        string
	targetVersion       string
	areErrorsSuppressed bool
	technologies        []string
	flavor              string

	log = logd.Get().WithName("bootstrap")
)

func New() *cobra.Command {
	fs := afero.NewOsFs()

	return newCmd(fs)
}

func newCmd(fs afero.Fs) *cobra.Command {
	cmd := &cobra.Command{
		Use:                use,
		RunE:               run(afero.Afero{Fs: fs}),
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		SilenceUsage:       true,
	}

	AddFlags(cmd)

	return cmd
}

func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&targetFolder, TargetFolderFlag, "", "Base path where to copy the codemodule to.")
	_ = cmd.MarkPersistentFlagRequired(TargetFolderFlag)

	cmd.PersistentFlags().StringVar(&targetVersion, TargetVersionFlag, "", "Version of the zip to be downloaded. If not set, CSI-driver mount is expected to be used.")

	cmd.PersistentFlags().BoolVar(&areErrorsSuppressed, SuppressErrorsFlag, false, "(Optional) Always return exit code 0, even on error")

	cmd.PersistentFlags().Lookup(SuppressErrorsFlag).NoOptDefVal = "true"

	cmd.PersistentFlags().StringSliceVar(&technologies, TechnologiesFlag, []string{"all"}, "comma separated list of technologies that will be used to download the code modules image.")

	cmd.PersistentFlags().StringVar(&flavor, FlavorFlag, arch.Flavor, "flavor of the code modules image.")

	configure.AddFlags(cmd)
}

func run(fs afero.Afero) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		unix.Umask(0000)

		signalHandler := ctrl.SetupSignalHandler()

		if targetVersion != "" {
			inputDir, _ := cmd.Flags().GetString(configure.InputFolderFlag)

			props := url.Properties{
				Os:            dtclient.OsUnix,
				Type:          dtclient.InstallerTypePaaS,
				Flavor:        flavor,
				Arch:          arch.Arch,
				Technologies:  technologies,
				TargetVersion: targetVersion,
				Url:           "",
				SkipMetadata:  false,
				PathResolver:  metadata.PathResolver{RootDir: consts.AgentBinDirMount}, // ?
			}

			client := download.New()

			err := client.Do(signalHandler, fs, inputDir, targetFolder, props)
			if err != nil {
				if areErrorsSuppressed {
					log.Error(err, "error during download, the error was suppressed")

					return nil
				}

				log.Error(err, "error during download")

				return err
			}
		}

		err := configure.Execute(log.Logger, fs, targetFolder)
		if err != nil {
			if areErrorsSuppressed {
				log.Error(err, "error during configuration, the error was suppressed")

				return nil
			}

			log.Error(err, "error during configuration")

			return err
		}

		return nil
	}
}
