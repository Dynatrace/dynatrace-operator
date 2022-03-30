package daemonset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/version"
)

const (
	DeploymentTypeApplicationMonitoring = "application_monitoring"
	DeploymentTypeFullStack             = "classic_fullstack"
	DeploymentTypeCloudNative           = "cloud_native_fullstack"
	DeploymentTypeHostMonitoring        = "host_monitoring"
)

func (dsInfo *builderInfo) arguments() []string {
	metadata := deploymentmetadata.NewDeploymentMetadata(dsInfo.clusterId, dsInfo.deploymentType)
	args := dsInfo.hostInjectSpec.Args
	args = dsInfo.appendProxyArg(args)
	args = dsInfo.appendNetworkZoneArg(args)
	args = appendOperatorVersionArg(args)
	args = append(args, metadata.AsArgs()...)
	return args
}

func appendOperatorVersionArg(args []string) []string {
	return append(args, "--set-host-property=OperatorVersion="+version.Version)
}

func (dsInfo *builderInfo) appendNetworkZoneArg(args []string) []string {
	if dsInfo.instance.Spec.NetworkZone != "" {
		return append(args, fmt.Sprintf("--set-network-zone=%s", dsInfo.instance.Spec.NetworkZone))
	}
	return args
}

func (dsInfo *builderInfo) appendProxyArg(args []string) []string {
	if dsInfo.hasProxy() {
		return append(args, "--set-proxy=$(https_proxy)")
	}
	return args
}

func (dsInfo *builderInfo) hasProxy() bool {
	return dsInfo.instance.Spec.Proxy != nil && (dsInfo.instance.Spec.Proxy.ValueFrom != "" || dsInfo.instance.Spec.Proxy.Value != "")
}
