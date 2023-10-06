package daemonset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/parametermap"
)

const argumentPrefix = "--"
const customArgumentPriority = 2
const defaultArgumentPriority = 1

func (dsInfo *builderInfo) arguments() []string {
	argMap := parametermap.NewMap(parametermap.WithSeparator(parametermap.DefaultSeparator), parametermap.WithPriority(defaultArgumentPriority))

	dsInfo.appendProxyArg(argMap)
	dsInfo.appendNetworkZoneArg(argMap)

	appendOperatorVersionArg(argMap)
	appendImmutableImageArgs(argMap)
	dsInfo.appendHostInjectArgs(argMap)

	return argMap.AsKeyValueStrings()
}

func appendImmutableImageArgs(argMap *parametermap.Map) {
	argMap.Append(argumentPrefix+"set-tenant", fmt.Sprintf("$(%s)", connectioninfo.EnvDtTenant))
	argMap.Append(argumentPrefix+"set-server", fmt.Sprintf("{$(%s)}", connectioninfo.EnvDtServer))
}

func (dsInfo *builderInfo) appendHostInjectArgs(argMap *parametermap.Map) {
	if dsInfo.hostInjectSpec != nil {
		parametermap.Append(argMap, dsInfo.hostInjectSpec.Args, parametermap.WithPriority(customArgumentPriority))
	}
}

func appendOperatorVersionArg(argMap *parametermap.Map) {
	argMap.Append(argumentPrefix+"set-host-property", fmt.Sprintf("OperatorVersion=$(%s)", deploymentmetadata.EnvDtOperatorVersion))
}

func (dsInfo *builderInfo) appendNetworkZoneArg(argMap *parametermap.Map) {
	if dsInfo.dynakube != nil && dsInfo.dynakube.Spec.NetworkZone != "" {
		argMap.Append(argumentPrefix+"set-network-zone", dsInfo.dynakube.Spec.NetworkZone)
	}
}

func (dsInfo *builderInfo) appendProxyArg(argMap *parametermap.Map) {
	if dsInfo.hasProxy() {
		argMap.Append(argumentPrefix+"set-proxy", "$(https_proxy)")
	}
	// if no proxy is set, we still have to set it as empty to clear proxy settings the OA might have cached
	argMap.Append(argumentPrefix+"set-proxy", "")
}

func (dsInfo *builderInfo) hasProxy() bool {
	return dsInfo.dynakube != nil && dsInfo.dynakube.NeedsOneAgentProxy()
}
