package standalone

import (
	"fmt"
	"path/filepath"
)

func (runner *Runner) getBaseConfContent(container containerInfo) string {
	formatString := `[container]
containerName %s
imageName %s
k8s_fullpodname %s
k8s_poduid %s
k8s_containername %s
k8s_basepodname %s
k8s_namespace %s
`
	return fmt.Sprintf(formatString,
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
	formatString := `k8s_node_name %s
k8s_cluster_id %s
`
	return fmt.Sprintf(formatString,
		runner.env.k8NodeName,
		runner.config.ClusterID,
	)
}

func (runner *Runner) getHostConfContent() string {
	formatString := `[host]
tenant %s
isCloudNativeFullStack true"
`
	return fmt.Sprintf(formatString,
		runner.hostTenant,
	)
}

func (runner *Runner) createJsonEnrichmentFile() error {
	jsonFormat := `"k8s.pod.uid": "%s",
"k8s.pod.name": "%s",
"k8s.namespace.name": "%s",
"dt.kubernetes.workload.kind": "%s",
"dt.kubernetes.workload.name": "%s",
"dt.kubernetes.cluster.id": "%s"
`
	jsonContent := fmt.Sprintf(jsonFormat,
		runner.env.k8PodUID,
		runner.env.k8PodName,
		runner.env.k8Namespace,
		runner.env.workloadKind,
		runner.env.workloadName,
		runner.config.ClusterID,
	)
	jsonPath := filepath.Join(EnrichmentPath, "dt_metadata.json")
	return createConfFile(jsonPath, jsonContent)

}

func (runner *Runner) createPropsEnrichmentFile() error {
	propsFormat := `k8s.pod.uid=%s
k8s.pod.name=%s
k8s.namespace.name=%s
dt.kubernetes.workload.kind=%s
dt.kubernetes.workload.name=%s
dt.kubernetes.cluster.id=%s
`
	propsContent := fmt.Sprintf(propsFormat,
		runner.env.k8PodUID,
		runner.env.k8PodName,
		runner.env.k8Namespace,
		runner.env.workloadKind,
		runner.env.workloadName,
		runner.config.ClusterID,
	)
	propsPath := filepath.Join(EnrichmentPath, "dt_metadata.properties")
	return createConfFile(propsPath, propsContent)
}

func createConfFile(path string, content string) error {
	// TODO: create conf file
	return nil
}
