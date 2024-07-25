package startup

import (
	"encoding/json"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
)

type enrichmentJson struct {
	PodUid        string `json:"k8s.pod.uid"`
	PodName       string `json:"k8s.pod.name"`
	NamespaceName string `json:"k8s.namespace.name"`
	ClusterName   string `json:"k8s.cluster.name,omitempty"`
	ClusterUID    string `json:"k8s.cluster.uid"`
	WorkloadKind  string `json:"k8s.workload.kind"`
	WorkloadName  string `json:"k8s.workload.name"`

	// Deprecated
	DTClusterEntity string `json:"dt.entity.kubernetes_cluster,omitempty"`
	// Deprecated
	DTClusterID string `json:"dt.kubernetes.cluster.id"`
	// Deprecated
	DTWorkloadKind string `json:"dt.kubernetes.workload.kind"`
	// Deprecated
	DTWorkloadName string `json:"dt.kubernetes.workload.name"`
}

var (
	enrichmentJsonPath  = filepath.Join(consts.EnrichmentInitPath, consts.EnrichmentJsonFilename)
	enrichmentPropsPath = filepath.Join(consts.EnrichmentInitPath, consts.EnrichmentPropertiesFilename)
)

func (runner *Runner) createEnrichmentFiles() error {
	data := enrichmentJson{
		PodUid:        runner.env.K8PodUID,
		PodName:       runner.env.K8PodName,
		NamespaceName: runner.env.K8Namespace,
		ClusterName:   runner.env.K8ClusterName,
		ClusterUID:    runner.env.K8ClusterID,
		WorkloadKind:  runner.env.WorkloadKind,
		WorkloadName:  runner.env.WorkloadName,

		DTClusterEntity: runner.env.K8ClusterName,
		DTClusterID:     runner.env.K8ClusterID,
		DTWorkloadKind:  runner.env.WorkloadKind,
		DTWorkloadName:  runner.env.WorkloadName,
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = runner.createConfigFile(enrichmentJsonPath, string(raw), true)
	if err != nil {
		return err
	}

	props := map[string]string{}

	err = json.Unmarshal(raw, &props)
	if err != nil {
		return err
	}

	propsContent := ""
	for key, value := range props {
		propsContent += key + "=" + value + "\n"
	}

	return runner.createConfigFile(enrichmentPropsPath, propsContent, true)
}
