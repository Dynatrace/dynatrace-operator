package validation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"
)

const (
	warningFeatureFlagDeprecated   = `Using deprecated feature flags: `
	warningFeatureFlagUnknown      = `Using unknown feature flags: %s. Please remove them from Dynakube specification.`
	warningNodeImagePullWithoutCSI = "The `" + exp.OANodeImagePullKey + "` annotation is set, but the CSI driver is not available on this cluster. This feature flag only affects the behavior of the CSI driver, so it will have no effect. Other previous `node-image-pull` related behavior has been defaulted."
	errorInvalidNoProxy            = "The DynaKube's specification has an invalid value set using the " + exp.NoProxyKey + " annotation. Make sure to remove forbidden characters (newline, tab, carriage return, null) from the value in your custom resource."
)

var deprecatedFeatureFlags = []string{
	exp.OAProxyIgnoredKey,   //nolint:staticcheck
	exp.AGUpdatesKey,        //nolint:staticcheck
	exp.AGDisableUpdatesKey, //nolint:staticcheck
	exp.OAMaxUnavailableKey, //nolint:staticcheck
	exp.AGK8sAppEnabledKey,  //nolint:staticcheck
}

var knownFeatureFlags = []string{
	// flag.go
	exp.NoProxyKey,
	exp.UseEECLegacyMountsKey,
	exp.UsePublicRegistryKey,
	// activegate.go
	exp.AGDisableUpdatesKey, //nolint:staticcheck
	exp.AGIgnoreProxyKey,    //nolint:staticcheck
	exp.AGUpdatesKey,        //nolint:staticcheck
	exp.AGAppArmorKey,
	exp.AGAutomaticK8sAPIMonitoringKey,
	exp.AGAutomaticK8sAPIMonitoringClusterNameKey,
	exp.AGK8sAppEnabledKey, //nolint:staticcheck
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
	exp.InjectionPullSecretKey,
	exp.InjectionSeccompKey, //nolint:staticcheck
	// oneagent.go
	exp.OAProxyIgnoredKey,   //nolint:staticcheck
	exp.OAMaxUnavailableKey, //nolint:staticcheck
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
		slices.Sort(results)

		return fmt.Sprintf(warningFeatureFlagUnknown, strings.Join(results, ", "))
	}

	return ""
}

func isNodeImagePullWithoutCSI(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if !dk.OneAgent().IsCSIAvailable() && dk.FF().IsSet(exp.OANodeImagePullKey) {
		return warningNodeImagePullWithoutCSI
	}

	return ""
}

func invalidNoProxy(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if strings.ContainsAny(dk.FF().GetNoProxy(), sanitize.InvalidCommandLineCharset) {
		return errorInvalidNoProxy
	}

	return ""
}
