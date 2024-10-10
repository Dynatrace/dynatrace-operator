package exp

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MultipleOsAgentsOnNodeAnnotation      = AnnotationPrefix + "multiple-osagents-on-node"
	OneAgentMaxUnavailableAnnotation      = AnnotationPrefix + "oneagent-max-unavailable"
	OneAgentInitialConnectRetryAnnotation = AnnotationPrefix + "oneagent-initial-connect-retry-ms"
	OneAgentPrivilegedAnnotation          = AnnotationPrefix + "oneagent-privileged"

	IgnoreUnknownStateAnnotation     = AnnotationPrefix + "ignore-unknown-state"
	IgnoredNamespacesAnnotation      = AnnotationPrefix + "ignored-namespaces"
	AutomaticInjectionAnnotation     = AnnotationPrefix + "automatic-injection"
	LabelVersionDetectionAnnotation  = AnnotationPrefix + "label-version-detection"
	InjectionFailurePolicyAnnotation = AnnotationPrefix + "injection-failure-policy"
	InitContainerSeccompAnnotation   = AnnotationPrefix + "init-container-seccomp-profile"
	EnforcementModeAnnotation        = AnnotationPrefix + "enforcement-mode"
	ReadOnlyOneAgentAnnotation       = AnnotationPrefix + "oneagent-readonly-host-fs"
)

var (
	IstioDefaultOneAgentInitialConnectRetry = 6000
)

// GetMaxUnavailableOneAgent is a feature flag to configure maxUnavailable on the OneAgent DaemonSets rolling upgrades.
func (f *FeatureFlags) GetMaxUnavailableOneAgent() int {
	return f.getIntValue(OneAgentMaxUnavailableAnnotation, 1)
}

// IsUnknownStateIgnored is a feature flag that makes the operator inject into applications even when the dynakube is in an UNKNOWN state,
// this may cause extra host to appear in the tenant for each process.
func (f *FeatureFlags) IsUnknownStateIgnored() bool {
	return f.getRawValue(IgnoreUnknownStateAnnotation) == truePhrase
}

// GetIgnoredNamespaces is a feature flag for ignoring certain namespaces.
// defaults to "[ \"^dynatrace$\", \"^kube-.*\", \"openshift(-.*)?\" ]".
func (f *FeatureFlags) GetIgnoredNamespaces(owner client.Object) []string {
	raw := f.getRawValue(IgnoredNamespacesAnnotation)
	if raw == "" {
		return f.getDefaultIgnoredNamespaces(owner)
	}

	ignoredNamespaces := &[]string{}

	err := json.Unmarshal([]byte(raw), ignoredNamespaces)
	if err != nil {
		log.Error(err, "failed to unmarshal ignoredNamespaces feature-flag")

		return f.getDefaultIgnoredNamespaces(owner)
	}

	return *ignoredNamespaces
}

func (f *FeatureFlags) getDefaultIgnoredNamespaces(owner client.Object) []string {
	defaultIgnoredNamespaces := []string{
		fmt.Sprintf("^%s$", owner.GetNamespace()),
		"^kube-.*",
		"^openshift(-.*)?",
		"^gke-.*",
		"^gmp-.*",
	}

	return defaultIgnoredNamespaces
}

func (f *FeatureFlags) IsOneAgentPrivileged() bool {
	return f.getRawValue(OneAgentPrivilegedAnnotation) == truePhrase
}

// IsAutomaticInjectionEnabled controls OneAgent is injected to pods in selected namespaces automatically ("automatic-injection=true" or flag not set)
// or if pods need to be opted-in one by one ("automatic-injection=false").
func (f *FeatureFlags) IsAutomaticInjectionEnabled() bool {
	return f.getRawValue(AutomaticInjectionAnnotation) != falsePhrase
}

// FF().IsOneAgentReadOnly controls whether the host agent is run in readonly mode.
// In Host Monitoring disabling readonly mode, also disables the use of a CSI volume.
// Not compatible with Classic Fullstack.
func (f *FeatureFlags) IsOneAgentReadOnly() bool {
	return f.getRawValue(ReadOnlyOneAgentAnnotation) != falsePhrase
}

// IsMultipleOsAgentsOnNodeEnabled is a feature flag to enable multiple osagents running on the same host.
func (f *FeatureFlags) IsMultipleOsAgentsOnNodeEnabled() bool {
	return f.getRawValue(MultipleOsAgentsOnNodeAnnotation) == truePhrase
}

// IsLabelVersionDetectionEnabled is a feature flag to enable injecting additional environment variables based on user labels.
func (f *FeatureFlags) IsLabelVersionDetectionEnabled() bool {
	return f.getRawValue(LabelVersionDetectionAnnotation) == truePhrase
}

// GetOneAgentInitialConnectRetry is a feature flag to configure startup delay of standalone agents.
func (f *FeatureFlags) GetOneAgentInitialConnectRetry(isIstioEnabled bool) int {
	defaultValue := -1
	ffValue := f.getIntValue(OneAgentInitialConnectRetryAnnotation, defaultValue)

	// In case of istio, we want to have a longer initial delay for codemodules to ensure the DT service is created consistently
	if ffValue == defaultValue && isIstioEnabled {
		ffValue = IstioDefaultOneAgentInitialConnectRetry
	}

	return ffValue
}

func (f *FeatureFlags) GetInjectionFailurePolicy() string {
	if f.getRawValue(InjectionFailurePolicyAnnotation) == failPhrase {
		return failPhrase
	}

	return silentPhrase
}

func (f *FeatureFlags) IsInitContainerSeccompEnabled() bool {
	return f.getRawValue(InitContainerSeccompAnnotation) == truePhrase
}

// FeatureEnforcementMode is a feature flag to control how the initContainer
// sets the tenantUUID to the container.conf file (always vs if oneAgent is present).
func (f *FeatureFlags) IsEnforcementModeEnabled() bool {
	return f.getRawValue(EnforcementModeAnnotation) != falsePhrase
}
