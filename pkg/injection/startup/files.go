package startup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const (
	onlyReadAllFileMode = 0444
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

	k8HostInfoFormatString = `k8s_node_name %s
[host]
tenant %s
isCloudNativeFullStack true
`

	k8ClusterIDFormatString = `k8s_cluster_id %s
`

	curlOptionsFormatString = `initialConnectRetryMs %d
`
)

func (runner *Runner) getBaseConfContent(container ContainerInfo) string {
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

func (runner *Runner) getK8SHostInfo() string {
	return fmt.Sprintf(k8HostInfoFormatString,
		runner.env.K8NodeName,
		runner.hostTenant,
	)
}

func (runner *Runner) getK8SClusterID() string {
	return fmt.Sprintf(k8ClusterIDFormatString,
		runner.env.K8ClusterID,
	)
}

func (runner *Runner) getCurlOptionsContent() string {
	return fmt.Sprintf(curlOptionsFormatString, runner.config.InitialConnectRetry)
}

func (runner *Runner) createCurlOptionsFile(container ContainerInfo) error {
	content := runner.getCurlOptionsContent()
	path := filepath.Join(consts.SharedDirMount, consts.AgentCurlOptionsFileName)

	return runner.createConfigFile(path, content, true)
}

func (runner *Runner) createConfigFile(path string, content string, verbose bool) error {
	err := createFile(runner.fs, path, content)
	if err != nil {
		return errors.WithStack(err)
	}

	if verbose {
		log.Info("created file", "filePath", path, "content", content)
	} else {
		log.Info("created file", "filePath", path)
	}

	return nil
}

func createFile(fs afero.Fs, path string, content string) error {
	err := fs.MkdirAll(filepath.Dir(path), onlyReadAllFileMode)
	if err != nil {
		return errors.WithStack(err)
	}

	file, err := fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, onlyReadAllFileMode)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = file.Write([]byte(content))
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
