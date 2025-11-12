package metadata

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	use = "generate-metadata"

	metadataFileFlagName = "file"
	attributesFlagName   = "attributes"
)

var (
	metadataFileFlagValue string
	attributesFlagValue   string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:          use,
		Short:        "Generates a metadata file",
		Long:         "Generates a metadata file containing attributes in key=value format. One attribute per line",
		RunE:         run(),
		SilenceUsage: true,
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&metadataFileFlagValue, metadataFileFlagName, "", "Path where the metadata file will be created")
	cmd.PersistentFlags().StringVar(&attributesFlagValue, attributesFlagName, "", "Comma-separated list of attributes to include")

	_ = cmd.MarkPersistentFlagRequired(metadataFileFlagName)
	_ = cmd.MarkPersistentFlagRequired(attributesFlagName)
}

func run() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log.Info("generating metadata file", "path", metadataFileFlagValue, "attributes", attributesFlagValue)

		content, err := parseAttributes(attributesFlagValue)
		if err != nil {
			log.Error(err, "failed to parse attributes")

			return err
		}

		err = writeMetadataFile(metadataFileFlagValue, content)
		if err != nil {
			log.Error(err, "failed to write metadata file")

			return err
		}

		log.Info("successfully generated metadata file")

		return nil
	}
}

func writeMetadataFile(filePath string, content string) error {
	dirPath := filepath.Dir(filePath)

	if dirPath != "" {
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// #nosec G306 -- metadata file is not sensitive, 0644 is intentional
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
