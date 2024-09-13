package startup

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
)

type enrichmentJson struct {
	ContainerName   string `json:"k8s.container.name"`
	PodUid          string `json:"k8s.pod.uid"`
	PodName         string `json:"k8s.pod.name"`
	NodeName        string `json:"k8s.node.name"`
	NamespaceName   string `json:"k8s.namespace.name"`
	ClusterName     string `json:"k8s.cluster.name,omitempty"`
	ClusterUID      string `json:"k8s.cluster.uid"`
	WorkloadKind    string `json:"k8s.workload.kind"`
	WorkloadName    string `json:"k8s.workload.name"`
	DTClusterEntity string `json:"dt.entity.kubernetes_cluster,omitempty"`

	// Deprecated
	DTClusterID string `json:"dt.kubernetes.cluster.id"`
	// Deprecated
	DTWorkloadKind string `json:"dt.kubernetes.workload.kind"`
	// Deprecated
	DTWorkloadName string `json:"dt.kubernetes.workload.name"`
}

var (
	enrichmentJsonPathTemplate  = filepath.Join(consts.EnrichmentInitPath, consts.EnrichmentInitJsonFilenameTemplate)
	enrichmentPropsPathTemplate = filepath.Join(consts.EnrichmentInitPath, consts.EnrichmentInitPropertiesFilenameTemplate)
)

func (runner *Runner) createEnrichmentFiles() error {
	for _, container := range runner.env.Containers {
		data := enrichmentJson{
			ContainerName: container.Name,
			PodUid:        runner.env.K8PodUID,
			PodName:       runner.env.K8PodName,
			NodeName:      runner.env.K8NodeName,
			NamespaceName: runner.env.K8Namespace,
			ClusterName:   runner.env.K8ClusterName,
			ClusterUID:    runner.env.K8ClusterID,
			WorkloadKind:  runner.env.WorkloadKind,
			WorkloadName:  runner.env.WorkloadName,

			DTClusterEntity: runner.env.K8ClusterEntityID,
			DTClusterID:     runner.env.K8ClusterID,
			DTWorkloadKind:  runner.env.WorkloadKind,
			DTWorkloadName:  runner.env.WorkloadName,
		}

		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}

		content := map[string]string{}

		err = json.Unmarshal(raw, &content)
		if err != nil {
			return err
		}

		for key, value := range runner.env.WorkloadAnnotations {
			content[key] = value
		}

		jsonContent, err := json.Marshal(content)
		if err != nil {
			return err
		}

		err = runner.createConfigFile(fmt.Sprintf(enrichmentJsonPathTemplate, container.Name), string(jsonContent), true)
		if err != nil {
			return err
		}

		var propsContent strings.Builder
		for key, value := range content {
			propsContent.WriteString(key)
			propsContent.WriteString("=")
			propsContent.WriteString(value)
			propsContent.WriteString("\n")
		}

		err = runner.createConfigFile(fmt.Sprintf(enrichmentPropsPathTemplate, container.Name), propsContent.String(), true)
		if err != nil {
			return err
		}
	}

	return nil
}
