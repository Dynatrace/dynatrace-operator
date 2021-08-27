package oneagent

import (
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/version"
)

func prepareArgs(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, feature string, clusterID string) []string {
	args := fs.Args
	if p := instance.Spec.Proxy; p != nil && (p.ValueFrom != "" || p.Value != "") {
		args = append(args, fmt.Sprintf("--set-proxy=${%s}", DTProxy))
	}

	if instance.Spec.NetworkZone != "" {
		args = append(args, fmt.Sprintf("--set-network-zone=%s", instance.Spec.NetworkZone))
	}

	if feature == InframonFeature {
		args = append(args, "--set-host-id-source=k8s-node-name")
	}

	args = append(args, "--set-host-property=OperatorVersion="+version.Version)

	metadata := deploymentmetadata.NewDeploymentMetadata(clusterID)
	args = append(args, metadata.AsArgs()...)
	return args
}
