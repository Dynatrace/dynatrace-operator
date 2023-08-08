package daemonset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
)

func (dsInfo *builderInfo) arguments() []string {
	args := make([]string, 0)

	args = dsInfo.appendProxyArg(args)
	args = dsInfo.appendNetworkZoneArg(args)
	args = appendOperatorVersionArg(args)
	args = appendImmutableImageArgs(args)
	args = dsInfo.appendHostInjectArgs(args)

	return args
}

func appendImmutableImageArgs(args []string) []string {
	args = append(args, fmt.Sprintf("--set-tenant=$(%s)", connectioninfo.EnvDtTenant))
	args = append(args, fmt.Sprintf("--set-server={$(%s)}", connectioninfo.EnvDtServer))
	return args
}

func (dsInfo *builderInfo) appendHostInjectArgs(args []string) []string {
	if dsInfo.hostInjectSpec != nil {
		return append(args, dsInfo.hostInjectSpec.Args...)
	}

	return args
}

func appendOperatorVersionArg(args []string) []string {
	return append(args, fmt.Sprintf("--set-host-property=OperatorVersion=$(%s)", deploymentmetadata.EnvDtOperatorVersion))
}

func (dsInfo *builderInfo) appendNetworkZoneArg(args []string) []string {
	if dsInfo.dynakube != nil && dsInfo.dynakube.Spec.NetworkZone != "" {
		return append(args, fmt.Sprintf("--set-network-zone=%s", dsInfo.dynakube.Spec.NetworkZone))
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
	return dsInfo.dynakube != nil && dsInfo.dynakube.NeedsOneAgentProxy()
}
