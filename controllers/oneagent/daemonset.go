package oneagent

import (
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/version"
)

func prepareArgs(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, feature string, clusterID string) []string {
	args := fs.Args
	if instance.Spec.Proxy != nil && (instance.Spec.Proxy.ValueFrom != "" || instance.Spec.Proxy.Value != "") {
		args = append(args, "--set-proxy=$(https_proxy)")
	}

	if instance.Spec.NetworkZone != "" {
		args = append(args, fmt.Sprintf("--set-network-zone=%s", instance.Spec.NetworkZone))
	}

	if feature == InframonFeature {
		args = append(args, "--set-host-id-source=k8s-node-name")
	} else {
		args = append(args, "--set-host-id-source=auto")
	}

	args = append(args, "--set-host-property=OperatorVersion="+version.Version)

	metadata := deploymentmetadata.NewDeploymentMetadata(clusterID, *instance)
	args = append(args, metadata.AsArgs()...)
	return args
}
