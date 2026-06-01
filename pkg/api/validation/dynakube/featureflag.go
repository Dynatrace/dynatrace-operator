package validation

import (
	"context"
	"slices"
	"sort"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningFeatureFlagDeprecated   = `Using deprecated feature flags: `
	warningFeatureFlagUnknown      = `Using unknown feature flags: `
	warningNodeImagePullWithoutCSI = "The `" + exp.OANodeImagePullKey + "` annotation is set, but the CSI driver is not available on this cluster. This feature flag only affects the behavior of the CSI driver, so it will have no effect. Other previous `node-image-pull` related behavior has been defaulted."
)

var deprecatedFeatureFlags = []string{
	exp.OAProxyIgnoredKey,   //nolint:staticcheck
	exp.AGUpdatesKey,        //nolint:staticcheck
	exp.AGDisableUpdatesKey, //nolint:staticcheck
	exp.OAMaxUnavailableKey, //nolint:staticcheck
}

var knownFeatureFlags = []string{
	// flag.go
	exp.NoProxyKey,
	exp.UseEECLegacyMountsKey,
	exp.UsePublicRegistryKey,
	// activegate.go
	exp.AGDisableUpdatesKey,
	exp.AGIgnoreProxyKey,
	exp.AGUpdatesKey,
	exp.AGAppArmorKey,
	exp.AGAutomaticK8sAPIMonitoringKey,
	exp.AGAutomaticK8sAPIMonitoringClusterNameKey,
	exp.AGK8sAppEnabledKey,
	exp.AGAutomaticTLSCertificateKey,
	// csi.go
	exp.CSIMaxFailedMountAttemptsKey,
	exp.CSIMaxMountTimeoutKey,
	// enrichment.go
	exp.EnrichmentEnableAttributesDTKubernetes,
	// injection.go
	exp.InjectionIgnoredNamespacesKey,
	exp.InjectionAutomaticKey,
	exp.InjectionLabelVersionDetectionKey,
	exp.InjectionFailurePolicyKey,
	exp.InjectionSeccompKey,
	// oneagent.go
	exp.OAProxyIgnoredKey,
	exp.OAMaxUnavailableKey,
	exp.OAInitialConnectRetryKey,
	exp.OAPrivilegedKey,
	exp.OASkipLivenessProbeKey,
	exp.OANodeImagePullKey,
	exp.OANodeImagePullTechnologiesKey,
	// otlp.go
	exp.OTLPInjectionSetNoProxy,
}

func deprecatedFeatureFlag(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	var results []string

	if len(dk.Annotations) == 0 {
		return ""
	}

	for _, flag := range deprecatedFeatureFlags {
		if dk.FF().IsSet(flag) {
			results = append(results, flag)
		}
	}

	if len(results) > 0 {
		return warningFeatureFlagDeprecated + strings.Join(results, ", ")
	}

	return ""
}

func unknownFeatureFlag(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	var results []string

	for annotation := range dk.Annotations {
		if !strings.HasPrefix(annotation, exp.FFPrefix) && annotation != exp.OANodeImagePullTechnologiesKey {
			continue
		}

		if !slices.Contains(knownFeatureFlags, annotation) {
			results = append(results, annotation)
		}
	}

	if len(results) > 0 {
		sort.Strings(results)

		return warningFeatureFlagUnknown + strings.Join(results, ", ")
	}

	return ""
}

func isNodeImagePullWithoutCSI(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if !dk.OneAgent().IsCSIAvailable() && dk.FF().IsSet(exp.OANodeImagePullKey) {
		return warningNodeImagePullWithoutCSI
	}

	return ""
}
