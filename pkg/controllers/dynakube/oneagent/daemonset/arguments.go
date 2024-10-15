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
		prioritymap.WithAllowDuplicates(),
	)

	isProxyAsEnvDeprecated, err := isProxyAsEnvVarDeprecated(b.dk.OneAgentVersion())
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

	if b.dk.ClassicFullStackMode() {
		argMap.Append(argumentPrefix+"set-host-id-source", classicHostIdSource)
	} else if b.dk.HostMonitoringMode() || b.dk.CloudNativeFullstackMode() {
		argMap.Append(argumentPrefix+"set-host-id-source", inframonHostIdSource)
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
	if b.dk.NeedsCustomNoProxy() {
		noProxyValue := b.dk.FeatureNoProxy()

		if b.dk.ActiveGate().IsEnabled() {
			multiCap := capability.NewMultiCapability(b.dk)
			noProxyActiveGateValue := capability.BuildDNSEntryPointWithoutEnvVars(b.dk.Name, b.dk.Namespace, multiCap)

			if noProxyValue != "" {
				noProxyValue = strings.Join([]string{noProxyValue, noProxyActiveGateValue}, ",")
			} else {
				noProxyValue = noProxyActiveGateValue
			}
		}

		argMap.Append(argumentPrefix+"set-no-proxy", noProxyValue)
	} else {
		// if no-proxy is not set, we still have to set it as empty to clear proxy settings the OA might have cached
		argMap.Append(argumentPrefix+"set-no-proxy", "")
	}
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
