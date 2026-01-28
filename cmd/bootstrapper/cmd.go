package bootstrapper

import (
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/move"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper/download"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	Use = "bootstrap"

	TargetFolderFlag   = k8sinit.TargetFolderFlag
	SuppressErrorsFlag = k8sinit.SuppressErrorsFlag
	TechnologiesFlag   = move.TechnologyFlag

	TargetVersionFlag      = "version"
	FlavorFlag             = "flavor"
	MetadataEnrichmentFlag = "metadata-enrichment"
)

var (
	targetFolder        string
	targetVersion       string
	areErrorsSuppressed bool
	technologies        []string
	flavor              string

	needsMetadataEnrichment bool

	log = logd.Get().WithName("bootstrap")
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:                Use,
		RunE:               run,
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		SilenceUsage:       true,
	}

	AddFlags(cmd)

	return cmd
}

func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&targetFolder, TargetFolderFlag, "", "Base path where to copy the codemodule to.")

	cmd.PersistentFlags().StringVar(&targetVersion, TargetVersionFlag, "", "Version of the zip to be downloaded. If not set, CSI-driver mount is expected to be used.")

	cmd.PersistentFlags().BoolVar(&areErrorsSuppressed, SuppressErrorsFlag, false, "(Optional) Always return exit code 0, even on error")

	cmd.PersistentFlags().Lookup(SuppressErrorsFlag).NoOptDefVal = "true"

	cmd.PersistentFlags().StringSliceVar(&technologies, TechnologiesFlag, []string{"all"}, "comma separated list of technologies that will be used to download the code modules image.")

	cmd.PersistentFlags().StringVar(&flavor, FlavorFlag, arch.Flavor, "flavor of the code modules image.")

	cmd.PersistentFlags().BoolVar(&needsMetadataEnrichment, MetadataEnrichmentFlag, false, "(Optional) Should the enrichment with metadata be performed.")

	cmd.PersistentFlags().Lookup(MetadataEnrichmentFlag).NoOptDefVal = "true"

	configure.AddFlags(cmd)
}

func run(cmd *cobra.Command, _ []string) error {
	unix.Umask(0000)

	if targetVersion != "" {
		inputDir, _ := cmd.Flags().GetString(configure.InputFolderFlag)

		props := url.Properties{
			Os:            dtclient.OsUnix,
			Type:          dtclient.InstallerTypePaaS,
			Flavor:        flavor,
			Arch:          arch.Arch,
			Technologies:  technologies,
			TargetVersion: targetVersion,
			URL:           "",
			SkipMetadata:  false,
			PathResolver:  metadata.PathResolver{RootDir: targetFolder},
		}

		client := download.New()

		signalHandler := ctrl.SetupSignalHandler()

		err := client.Do(signalHandler, inputDir, targetFolder, props)
		if err != nil {
			if areErrorsSuppressed {
				log.Error(err, "error during download, the error was suppressed")

				return nil
			}

			log.Info("error during download")

			return err
		}
	}

	err := runConfigure()
	if err != nil {
		if areErrorsSuppressed {
			log.Error(err, "error during configuration, the error was suppressed")

			return nil
		}

		return err
	}

	return nil
}

func runConfigure() error {
	if targetFolder != "" {
		err := configure.SetupOneAgent(log.Logger, targetFolder)
		if err != nil {
			log.Info("error during oneagent configuration")

			return err
		}
	}

	if needsMetadataEnrichment {
		err := configure.EnrichWithMetadata(log.Logger)
		if err != nil {
			log.Info("error during metadata enrichment")

			return err
		}
	}

	return nil
}
