package standalone

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

var (
	jsonEnrichmentContentFormatString = `"k8s.pod.uid": "%s",
	"k8s.pod.name": "%s",
	"k8s.namespace.name": "%s",
	"dt.kubernetes.workload.kind": "%s",
	"dt.kubernetes.workload.name": "%s",
	"dt.kubernetes.cluster.id": "%s"
	`

	propsEnrichmentContentFormatString = `k8s.pod.uid=%s
	k8s.pod.name=%s
	k8s.namespace.name=%s
	dt.kubernetes.workload.kind=%s
	dt.kubernetes.workload.name=%s
	dt.kubernetes.cluster.id=%s
	`
)

type dataIngestSetup struct {
	fs  afero.Fs
	env *environment
}

func newDataIngestSetup(fs afero.Fs, env *environment) *dataIngestSetup {
	return &dataIngestSetup{
		fs:  fs,
		env: env,
	}
}

func (setup *dataIngestSetup) setup() error {
	return setup.enrichMetadata()
}

func (setup *dataIngestSetup) enrichMetadata() error {
	if err := setup.createPropsEnrichmentFile(); err != nil {
		return err
	}
	if err := setup.createJsonEnrichmentFile(); err != nil {
		return err
	}
	return nil
}

func (setup *dataIngestSetup) createJsonEnrichmentFile() error {
	jsonContent := fmt.Sprintf(jsonEnrichmentContentFormatString,
		setup.env.K8PodUID,
		setup.env.K8PodName,
		setup.env.K8Namespace,
		setup.env.WorkloadKind,
		setup.env.WorkloadName,
		setup.env.K8ClusterID,
	)
	jsonPath := filepath.Join(EnrichmentPath, fmt.Sprintf(enrichmentFilenameTemplate, "json"))

	return errors.WithStack(createConfFile(setup.fs, jsonPath, jsonContent))

}

func (setup *dataIngestSetup) createPropsEnrichmentFile() error {
	propsContent := fmt.Sprintf(propsEnrichmentContentFormatString,
		setup.env.K8PodUID,
		setup.env.K8PodName,
		setup.env.K8Namespace,
		setup.env.WorkloadKind,
		setup.env.WorkloadName,
		setup.env.K8ClusterID,
	)
	propsPath := filepath.Join(EnrichmentPath, fmt.Sprintf(enrichmentFilenameTemplate, "properties"))

	return errors.WithStack(createConfFile(setup.fs, propsPath, propsContent))
}
