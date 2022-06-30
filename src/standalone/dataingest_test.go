package standalone

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestEnrichMetadata(t *testing.T) {
	runner := createTestDataIngestSetup(t)
	t.Run(`create enrichment files`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		assertIfEnrichmentFilesNotExists(t, *runner)
		err := runner.enrichMetadata()

		require.NoError(t, err)
		assertIfEnrichmentFilesExists(t, *runner)
	})
}

func assertIfEnrichmentFilesExists(t *testing.T, setup dataIngestSetup) {
	assertIfFileExists(t,
		setup.fs,
		filepath.Join(
			EnrichmentPath,
			fmt.Sprintf(enrichmentFilenameTemplate, "json")))
	assertIfFileExists(t,
		setup.fs,
		filepath.Join(
			EnrichmentPath,
			fmt.Sprintf(enrichmentFilenameTemplate, "properties")))

}

func assertIfEnrichmentFilesNotExists(t *testing.T, setup dataIngestSetup) {
	assertIfFileNotExists(t,
		setup.fs,
		filepath.Join(
			EnrichmentPath,
			fmt.Sprintf(enrichmentFilenameTemplate, "json")))
	assertIfFileNotExists(t,
		setup.fs,
		filepath.Join(
			EnrichmentPath,
			fmt.Sprintf(enrichmentFilenameTemplate, "properties")))

}

func createTestDataIngestSetup(t *testing.T) *dataIngestSetup {
	fs := prepTestFs(t)
	resetEnv := prepTestEnv(t)

	env, err := newEnv()
	require.NoError(t, err)

	setup := newDataIngestSetup(fs, env)
	resetEnv()
	require.NotNil(t, setup)
	return setup
}
