package daemonset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
)

const argumentPrefix = "--"
const customArgumentPriority = 2
const defaultArgumentPriority = 1

func (dsInfo *builderInfo) arguments() ([]string, error) {
	argMap := prioritymap.New(prioritymap.WithSeparator(prioritymap.DefaultSeparator), prioritymap.WithPriority(defaultArgumentPriority))

	isProxyAsEnvDeprecated, err := isProxyAsEnvVarDeprecated(dsInfo.dynakube.OneAgentVersion())
	if err != nil {
		return []string{}, err
	}

	if !isProxyAsEnvDeprecated {
		// deprecated
		dsInfo.appendProxyArg(argMap)
	}

	dsInfo.appendNetworkZoneArg(argMap)

	appendOperatorVersionArg(argMap)
	appendImmutableImageArgs(argMap)

	if dsInfo.dynakube.ClassicFullStackMode() {
		argMap.Append(argumentPrefix+"set-host-id-source", classicHostIdSource)
	} else if dsInfo.dynakube.HostMonitoringMode() || dsInfo.dynakube.CloudNativeFullstackMode() {
		argMap.Append(argumentPrefix+"set-host-id-source", inframonHostIdSource)
	}

	dsInfo.appendHostInjectArgs(argMap)

	if dsInfo.dynakube.CloudNativeFullstackMode() {
		dsInfo.appendHostGroupArg(argMap)
	}

	return argMap.AsKeyValueStrings(), nil
}

func appendImmutableImageArgs(argMap *prioritymap.Map) {
	argMap.Append(argumentPrefix+"set-tenant", fmt.Sprintf("$(%s)", connectioninfo.EnvDtTenant))
	argMap.Append(argumentPrefix+"set-server", fmt.Sprintf("{$(%s)}", connectioninfo.EnvDtServer))
}

func (dsInfo *builderInfo) appendHostInjectArgs(argMap *prioritymap.Map) {
	if dsInfo.hostInjectSpec != nil {
		prioritymap.Append(argMap, dsInfo.hostInjectSpec.Args, prioritymap.WithPriority(customArgumentPriority))
	}
}

func appendOperatorVersionArg(argMap *prioritymap.Map) {
	argMap.Append(argumentPrefix+"set-host-property", fmt.Sprintf("OperatorVersion=$(%s)", deploymentmetadata.EnvDtOperatorVersion))
}

func (dsInfo *builderInfo) appendNetworkZoneArg(argMap *prioritymap.Map) {
	if dsInfo.dynakube != nil && dsInfo.dynakube.Spec.NetworkZone != "" {
		argMap.Append(argumentPrefix+"set-network-zone", dsInfo.dynakube.Spec.NetworkZone)
	}
}

func (dsInfo *builderInfo) appendHostGroupArg(argMap *prioritymap.Map) {
	if dsInfo.dynakube != nil && dsInfo.dynakube.Spec.OneAgent.HostGroup != "" {
		argMap.Append(argumentPrefix+"set-host-group", dsInfo.dynakube.Spec.OneAgent.HostGroup, prioritymap.WithPriority(prioritymap.HighPriority))
	}
}

// deprecated
func (dsInfo *builderInfo) appendProxyArg(argMap *prioritymap.Map) {
	if dsInfo.hasProxy() {
		argMap.Append(argumentPrefix+"set-proxy", "$(https_proxy)")
	}
	// if no proxy is set, we still have to set it as empty to clear proxy settings the OA might have cached
	argMap.Append(argumentPrefix+"set-proxy", "")
}

// deprecated
func (dsInfo *builderInfo) hasProxy() bool {
	return dsInfo.dynakube != nil && dsInfo.dynakube.NeedsOneAgentProxy()
}
