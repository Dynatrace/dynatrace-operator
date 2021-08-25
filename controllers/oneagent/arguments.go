package oneagent

import (
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/version"
)

func prepareArgs(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, feature string, clusterID string) []string {
	args := fs.Args
	args = appendProxyArg(instance, args)
	args = appendNetworkZoneArg(instance, args)
	args = appendHostIdSourceArg(feature, args)
	args = appendOperatorVersionArg(args)

	dt := deploymentmetadata.DeploymentTypeFS
	if feature == InframonFeature {
		dt = deploymentmetadata.DeploymentTypeIS
	}

	metadata := deploymentmetadata.NewDeploymentMetadata(clusterID, dt)
	args = append(args, metadata.AsArgs()...)
	return args
}

func appendOperatorVersionArg(args []string) []string {
	return append(args, "--set-host-property=OperatorVersion="+version.Version)
}

func appendHostIdSourceArg(feature string, args []string) []string {
	if feature == InframonFeature {
		return append(args, "--set-host-id-source=k8s-node-name")
	}
	return append(args, "--set-host-id-source=auto")
}

func appendNetworkZoneArg(instance *dynatracev1alpha1.DynaKube, args []string) []string {
	if instance.Spec.NetworkZone != "" {
		return append(args, fmt.Sprintf("--set-network-zone=%s", instance.Spec.NetworkZone))
	}
	return args
}

func appendProxyArg(instance *dynatracev1alpha1.DynaKube, args []string) []string {
	if instance.Spec.Proxy != nil && (instance.Spec.Proxy.ValueFrom != "" || instance.Spec.Proxy.Value != "") {
		return append(args, "--set-proxy=$(https_proxy)")
	}
	return args
}
