package standalone

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
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
	k8ConfContentFormatString = `k8s_node_name %s
k8s_cluster_id %s
`
	hostConfContentFormatString = `[host]
tenant %s
isCloudNativeFullStack true
`
	curlOptionsFormatString = `initialConnectRetryMs %d
`
)

func (runner *oneAgentSetup) getBaseConfContent(container containerInfo) string {
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

func (runner *oneAgentSetup) getK8ConfContent() string {
	return fmt.Sprintf(k8ConfContentFormatString,
		runner.env.K8NodeName,
		runner.config.ClusterID,
	)
}

func (runner *oneAgentSetup) getHostConfContent() string {
	return fmt.Sprintf(hostConfContentFormatString,
		runner.hostTenant,
	)
}

func (runner *oneAgentSetup) getCurlOptionsContent() string {
	return fmt.Sprintf(curlOptionsFormatString, runner.config.InitialConnectRetry)
}

func (runner *oneAgentSetup) createCurlOptionsFile() error {
	content := runner.getCurlOptionsContent()
	path := filepath.Join(ShareDirMount, CurlOptionsFileName)

	return errors.WithStack(createConfFile(runner.fs, path, content))
}

func (runner *oneAgentSetup) setLDPreload() error {
	return createConfFile(runner.fs, filepath.Join(ShareDirMount, ldPreloadFilename), fmt.Sprintf("%s/agent/lib64/liboneagentproc.so", runner.env.InstallPath))
}

func (runner *oneAgentSetup) createContainerConfigurationFiles() error {
	for _, container := range runner.env.Containers {
		log.Info("creating conf file for container", "container", container)
		confFilePath := filepath.Join(ShareDirMount, fmt.Sprintf(ContainerConfFilenameTemplate, container.Name))
		content := runner.getBaseConfContent(container)
		if runner.hostTenant != NoHostTenant {
			if runner.config.TenantUUID == runner.hostTenant {
				log.Info("adding k8s fields")
				content += runner.getK8ConfContent()
			}
			log.Info("adding hostTenant field")
			content += runner.getHostConfContent()
		}
		if err := createConfFile(runner.fs, confFilePath, content); err != nil {
			return err
		}
	}
	return nil
}

func (runner *oneAgentSetup) propagateTLSCert() error {
	return createConfFile(runner.fs, filepath.Join(ShareDirMount, "custom.pem"), runner.config.TlsCert)
}
