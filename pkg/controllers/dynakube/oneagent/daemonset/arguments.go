package daemonset

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
)

const argumentPrefix = "--"
const customArgumentPriority = 2
const defaultArgumentPriority = 1

func (b *builder) arguments() ([]string, error) {
	argMap := prioritymap.New(
		prioritymap.WithSeparator(prioritymap.DefaultSeparator),
		prioritymap.WithPriority(defaultArgumentPriority),
		prioritymap.WithAvoidDuplicates(),
		prioritymap.WithAllowDuplicatesFor("--set-host-property"),
		prioritymap.WithAllowDuplicatesFor("--set-host-tag"),
	)

	isProxyAsEnvDeprecated, err := isProxyAsEnvVarDeprecated(b.dk.OneAgent().GetVersion())
	if err != nil {
		return []string{}, err
	}

	// the !NeedsOneAgentProxy check is needed, if no proxy is set, we still have to set it as empty to clear proxy settings the OA might have cached
	if !isProxyAsEnvDeprecated || !b.dk.NeedsOneAgentProxy() {
		// deprecated
		b.appendProxyArg(argMap)
	}

	b.appendNoProxyArg(argMap)
	b.appendNetworkZoneArg(argMap)

	appendOperatorVersionArg(argMap)
	appendImmutableImageArgs(argMap)

	if b.dk.OneAgent().IsClassicFullStackMode() {
		argMap.Append(argumentPrefix+"set-host-id-source", classicHostIDSource)
	} else if b.dk.OneAgent().IsHostMonitoringMode() || b.dk.OneAgent().IsCloudNativeFullstackMode() {
		argMap.Append(argumentPrefix+"set-host-id-source", inframonHostIDSource)
	}

	b.appendHostInjectArgs(argMap)

	b.appendHostGroupArg(argMap)

	return argMap.AsKeyValueStrings(), nil
}

func appendImmutableImageArgs(argMap *prioritymap.Map) {
	argMap.Append(argumentPrefix+"set-tenant", fmt.Sprintf("$(%s)", connectioninfo.EnvDtTenant))
	argMap.Append(argumentPrefix+"set-server", fmt.Sprintf("{$(%s)}", connectioninfo.EnvDtServer))
}

func (b *builder) appendHostInjectArgs(argMap *prioritymap.Map) {
	if b.hostInjectSpec != nil {
		prioritymap.Append(argMap, b.hostInjectSpec.Args, prioritymap.WithPriority(customArgumentPriority))
	}
}

func appendOperatorVersionArg(argMap *prioritymap.Map) {
	argMap.Append(argumentPrefix+"set-host-property", fmt.Sprintf("OperatorVersion=$(%s)", deploymentmetadata.EnvDtOperatorVersion))
}

func (b *builder) appendNetworkZoneArg(argMap *prioritymap.Map) {
	if b.dk != nil && b.dk.Spec.NetworkZone != "" {
		argMap.Append(argumentPrefix+"set-network-zone", b.dk.Spec.NetworkZone)
	}
}

func (b *builder) appendHostGroupArg(argMap *prioritymap.Map) {
	if b.dk != nil && b.dk.Spec.OneAgent.HostGroup != "" {
		argMap.Append(argumentPrefix+"set-host-group", b.dk.Spec.OneAgent.HostGroup, prioritymap.WithPriority(prioritymap.HighPriority))
	}
}

func (b *builder) appendNoProxyArg(argMap *prioritymap.Map) {
	var noProxyValue string

	if b.dk.NeedsCustomNoProxy() {
		noProxyValue = b.dk.FF().GetNoProxy()
	}

	if b.dk.ActiveGate().IsEnabled() {
		noProxyActiveGateValue := capability.BuildHostEntries(*b.dk)
		if noProxyValue != "" {
			noProxyValue = strings.Join([]string{noProxyValue, noProxyActiveGateValue}, ",")
		} else {
			noProxyValue = noProxyActiveGateValue
		}
	}

	if b.dk.FF().GetComponentNoProxy() != "" {
		before := noProxyValue
		noProxyValue = b.dk.FF().GetComponentNoProxy()
		log.Info("dff used component-no-proxy", "component-no-proxy", noProxyValue, "before", before)
	}

	argMap.Append(argumentPrefix+"set-no-proxy", noProxyValue)
}

// deprecated
func (b *builder) appendProxyArg(argMap *prioritymap.Map) {
	if b.hasProxy() {
		argMap.Append(argumentPrefix+"set-proxy", "$(https_proxy)")
	}
	// if no proxy is set, we still have to set it as empty to clear proxy settings the OA might have cached
	argMap.Append(argumentPrefix+"set-proxy", "")
}

// deprecated
func (b *builder) hasProxy() bool {
	return b.dk != nil && b.dk.NeedsOneAgentProxy()
}
