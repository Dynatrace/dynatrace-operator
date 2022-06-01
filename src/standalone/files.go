package standalone

import (
	"fmt"
	"github.com/pkg/errors"
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

	curlOptionsFormatString = `initialConnectRetryMs %d
`
)

func (runner *Runner) getBaseConfContent(container containerInfo) string {
	return fmt.Sprintf(baseConfContentFormatString,
		container.Name,
		container.Image,
		runner.env.K8PodName,
		runner.env.K8PodUID,
		container.Name,
		runner.env.K8BasePodName,
		runner.env.K8Namespace,
	)
}

func (runner *Runner) getK8ConfContent() string {
	return fmt.Sprintf(k8ConfContentFormatString,
		runner.env.K8NodeName,
		runner.config.ClusterID,
	)
}

func (runner *Runner) getHostConfContent() string {
	return fmt.Sprintf(hostConfContentFormatString,
		runner.hostTenant,
	)
}

func (runner *Runner) getCurlOptionsContent() string {
	return fmt.Sprintf(curlOptionsFormatString, runner.config.InitialConnectRetry)
}

func (runner *Runner) createJsonEnrichmentFile() error {
	jsonContent := fmt.Sprintf(jsonEnrichmentContentFormatString,
		runner.env.K8PodUID,
		runner.env.K8PodName,
		runner.env.K8Namespace,
		runner.env.WorkloadKind,
		runner.env.WorkloadName,
		runner.config.ClusterID,
	)
	jsonPath := filepath.Join(EnrichmentPath, fmt.Sprintf(enrichmentFilenameTemplate, "json"))

	return errors.WithStack(runner.createConfFile(jsonPath, jsonContent))

}

func (runner *Runner) createPropsEnrichmentFile() error {
	propsContent := fmt.Sprintf(propsEnrichmentContentFormatString,
		runner.env.K8PodUID,
		runner.env.K8PodName,
		runner.env.K8Namespace,
		runner.env.WorkloadKind,
		runner.env.WorkloadName,
		runner.config.ClusterID,
	)
	propsPath := filepath.Join(EnrichmentPath, fmt.Sprintf(enrichmentFilenameTemplate, "properties"))

	return errors.WithStack(runner.createConfFile(propsPath, propsContent))
}

func (runner *Runner) createCurlOptionsFile() error {
	content := runner.getCurlOptionsContent()
	path := filepath.Join(ShareDirMount, curlOptionsFileName)

	return errors.WithStack(runner.createConfFile(path, content))
}

func (runner *Runner) createConfFile(path string, content string) error {
	err := runner.fs.MkdirAll(filepath.Dir(path), 0770)
	if err != nil {
		return err
	}

	file, err := runner.fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	if err != nil {
		return err
	}

	_, err = file.Write([]byte(content))
	if err != nil {
		return err
	}

	log.Info("created file", "filePath", path, "content", content)
	return nil
}
