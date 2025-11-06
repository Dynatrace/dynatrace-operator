package nodemetadata

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	use = "generate-node-metadata"

	nodeMetadataFileFlagName = "node-metadata-file"
	nodeAttributesFlagName   = "node-attributes"
)

var (
	nodeMetadataFileFlagValue string
	nodeAttributesFlagValue   string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:          use,
		Short:        "Generate node-level metadata file",
		Long:         "Generates a properties file containing Kubernetes node-level metadata attributes",
		RunE:         run(),
		SilenceUsage: true,
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&nodeMetadataFileFlagValue, nodeMetadataFileFlagName, "", "Path where the node metadata file will be created")
	cmd.PersistentFlags().StringVar(&nodeAttributesFlagValue, nodeAttributesFlagName, "", "Comma-separated list of node attributes to include")

	_ = cmd.MarkPersistentFlagRequired(nodeMetadataFileFlagName)
	_ = cmd.MarkPersistentFlagRequired(nodeAttributesFlagName)
}

func run() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log.Info("generating node metadata file", "path", nodeMetadataFileFlagValue, "attributes", nodeAttributesFlagValue)

		content, err := parseAttributes(nodeAttributesFlagValue)
		if err != nil {
			log.Error(err, "failed to parse node attributes")

			return err
		}

		err = writeMetadataFile(nodeMetadataFileFlagValue, content)
		if err != nil {
			log.Error(err, "failed to write metadata file")

			return err
		}

		log.Info("successfully generated node metadata file")

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

	// #nosec G306 -- node metadata file is not sensitive, 0644 is intentional
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
