package standalone

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	baseConfContentFormatString = `[container]
containerName %s
imageName %s
k8s_fullpodname %s
k8s_poduid %s
k8s_containername %s
k8s_basepodname %s
k8s_namespace %s
`
	k8ConfContentFormatString = `k8s_node_name %s
k8s_cluster_id %s
`
	hostConfContentFormatString = `[host]
tenant %s
isCloudNativeFullStack true"
`

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

func (runner *Runner) getBaseConfContent(container containerInfo) string {
	return fmt.Sprintf(baseConfContentFormatString,
		container.name,
		container.image,
		runner.env.k8PodName,
		runner.env.k8PodUID,
		container.name,
		runner.env.k8BasePodName,
		runner.env.k8Namespace,
	)
}

func (runner *Runner) getK8ConfContent() string {
	return fmt.Sprintf(k8ConfContentFormatString,
		runner.env.k8NodeName,
		runner.config.ClusterID,
	)
}

func (runner *Runner) getHostConfContent() string {
	return fmt.Sprintf(hostConfContentFormatString,
		runner.hostTenant,
	)
}

func (runner *Runner) createJsonEnrichmentFile() error {
	jsonContent := fmt.Sprintf(jsonEnrichmentContentFormatString,
		runner.env.k8PodUID,
		runner.env.k8PodName,
		runner.env.k8Namespace,
		runner.env.workloadKind,
		runner.env.workloadName,
		runner.config.ClusterID,
	)
	jsonPath := filepath.Join(EnrichmentPath, fmt.Sprintf(enrichmentFilenameTemplate, "json"))
	return runner.createConfFile(jsonPath, jsonContent)

}

func (runner *Runner) createPropsEnrichmentFile() error {
	propsContent := fmt.Sprintf(propsEnrichmentContentFormatString,
		runner.env.k8PodUID,
		runner.env.k8PodName,
		runner.env.k8Namespace,
		runner.env.workloadKind,
		runner.env.workloadName,
		runner.config.ClusterID,
	)
	propsPath := filepath.Join(EnrichmentPath, fmt.Sprintf(enrichmentFilenameTemplate, "properties"))
	return runner.createConfFile(propsPath, propsContent)
}

func (runner *Runner) createConfFile(path string, content string) error {
	if err := runner.fs.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return err
	}
	file, err := runner.fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	if err != nil {
		return err
	}
	if _, err := file.Write([]byte(content)); err != nil {
		return err
	}
	return nil
}
