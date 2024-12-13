package oneagent

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OneAgentTenantSecretSuffix            = "-oneagent-tenant-secret"
	OneAgentConnectionInfoConfigMapSuffix = "-oneagent-connection-info"
	PodNameOsAgent                        = "oneagent"
	DefaultOneAgentImageRegistrySubPath   = "/linux/oneagent"
)

func NewOneAgent(spec *Spec, status *Status, codeModulesStatus *CodeModulesStatus, //nolint:revive
	name, apiUrlHost string,
	featureOneAgentPrivileged, featureOneAgentSkipLivenessProbe bool) *OneAgent {
	return &OneAgent{
		Spec:              spec,
		Status:            status,
		CodeModulesStatus: codeModulesStatus,

		name:       name,
		apiUrlHost: apiUrlHost,

		featureOneAgentPrivileged:        featureOneAgentPrivileged,
		featureOneAgentSkipLivenessProbe: featureOneAgentSkipLivenessProbe,
	}
}

func (oa *OneAgent) IsCSIAvailable() bool {
	return installconfig.GetModules().CSIDriver
}

// ApplicationMonitoringMode returns true when application only section is used.
func (oa *OneAgent) ApplicationMonitoringMode() bool {
	return oa.ApplicationMonitoring != nil
}

// CloudNativeFullstackMode returns true when cloud native fullstack section is used.
func (oa *OneAgent) CloudNativeFullstackMode() bool {
	return oa.CloudNativeFullStack != nil
}

// HostMonitoringMode returns true when host monitoring section is used.
func (oa *OneAgent) HostMonitoringMode() bool {
	return oa.HostMonitoring != nil
}

// ClassicFullStackMode returns true when classic fullstack section is used.
func (oa *OneAgent) ClassicFullStackMode() bool {
	return oa.ClassicFullStack != nil
}

// NeedsOneAgent returns true when a feature requires OneAgent instances.
func (oa *OneAgent) NeedsOneAgent() bool {
	return oa.ClassicFullStackMode() || oa.CloudNativeFullstackMode() || oa.HostMonitoringMode()
}

func (oa *OneAgent) OneAgentDaemonsetName() string {
	return fmt.Sprintf("%s-%s", oa.name, PodNameOsAgent)
}

func (oa *OneAgent) NeedsOneAgentPrivileged() bool {
	return oa.featureOneAgentPrivileged
}

func (oa *OneAgent) NeedsOneAgentReadinessProbe() bool {
	return oa.Healthcheck != nil
}

func (oa *OneAgent) NeedsOneAgentLivenessProbe() bool {
	return oa.Healthcheck != nil && !oa.featureOneAgentSkipLivenessProbe
}

// ShouldAutoUpdateOneAgent returns true if the Operator should update OneAgent instances automatically.
func (oa *OneAgent) ShouldAutoUpdateOneAgent() bool {
	switch {
	case oa.CloudNativeFullstackMode():
		return oa.CloudNativeFullStack.AutoUpdate == nil || *oa.CloudNativeFullStack.AutoUpdate
	case oa.HostMonitoringMode():
		return oa.HostMonitoring.AutoUpdate == nil || *oa.HostMonitoring.AutoUpdate
	case oa.ClassicFullStackMode():
		return oa.ClassicFullStack.AutoUpdate == nil || *oa.ClassicFullStack.AutoUpdate
	default:
		return false
	}
}

// OneagentTenantSecret returns the name of the secret containing the token for the OneAgent.
func (oa *OneAgent) OneagentTenantSecret() string {
	return oa.name + OneAgentTenantSecretSuffix
}

func (oa *OneAgent) OneAgentConnectionInfoConfigMapName() string {
	return oa.name + OneAgentConnectionInfoConfigMapSuffix
}

func (oa *OneAgent) UseReadOnlyOneAgents() bool {
	return oa.CloudNativeFullstackMode() || (oa.HostMonitoringMode() && oa.IsCSIAvailable())
}

func (oa *OneAgent) NeedAppInjection() bool {
	return oa.CloudNativeFullstackMode() || oa.ApplicationMonitoringMode()
}

func (oa *OneAgent) InitResources() *corev1.ResourceRequirements {
	if oa.ApplicationMonitoringMode() {
		return oa.ApplicationMonitoring.InitResources
	} else if oa.CloudNativeFullstackMode() {
		return oa.CloudNativeFullStack.InitResources
	}

	return nil
}

func (oa *OneAgent) OneAgentNamespaceSelector() *metav1.LabelSelector {
	switch {
	case oa.CloudNativeFullstackMode():
		return &oa.CloudNativeFullStack.NamespaceSelector
	case oa.ApplicationMonitoringMode():
		return &oa.ApplicationMonitoring.NamespaceSelector
	}

	return nil
}

func (oa *OneAgent) OneAgentSecCompProfile() string {
	switch {
	case oa.CloudNativeFullstackMode():
		return oa.CloudNativeFullStack.SecCompProfile
	case oa.HostMonitoringMode():
		return oa.HostMonitoring.SecCompProfile
	case oa.ClassicFullStackMode():
		return oa.ClassicFullStack.SecCompProfile
	default:
		return ""
	}
}

func (oa *OneAgent) OneAgentNodeSelector(fallbackNodeSelector map[string]string) map[string]string {
	switch {
	case oa.ClassicFullStackMode():
		return oa.ClassicFullStack.NodeSelector
	case oa.HostMonitoringMode():
		return oa.HostMonitoring.NodeSelector
	case oa.CloudNativeFullstackMode():
		return oa.CloudNativeFullStack.NodeSelector
	}

	return fallbackNodeSelector
}

// CustomCodeModulesImage provides the image reference for the CodeModules provided in the Spec.
func (oa *OneAgent) CustomCodeModulesImage() string {
	if oa.CloudNativeFullstackMode() {
		return oa.CloudNativeFullStack.CodeModulesImage
	} else if oa.ApplicationMonitoringMode() && oa.IsCSIAvailable() {
		return oa.ApplicationMonitoring.CodeModulesImage
	}

	return ""
}

// CustomCodeModulesVersion provides the version for the CodeModules provided in the Spec.
func (oa *OneAgent) CustomCodeModulesVersion() string {
	return oa.CustomOneAgentVersion()
}

// OneAgentImage provides the image reference set in Status for the OneAgent.
// Format: repo@sha256:digest.
func (oa *OneAgent) OneAgentImage() string {
	return oa.Status.ImageID
}

// OneAgentVersion provides version set in Status for the OneAgent.
func (oa *OneAgent) OneAgentVersion() string {
	return oa.Status.Version
}

// CustomOneAgentVersion provides the version for the OneAgent provided in the Spec.
func (oa *OneAgent) CustomOneAgentVersion() string {
	switch {
	case oa.ClassicFullStackMode():
		return oa.ClassicFullStack.Version
	case oa.CloudNativeFullstackMode():
		return oa.CloudNativeFullStack.Version
	case oa.ApplicationMonitoringMode():
		return oa.ApplicationMonitoring.Version
	case oa.HostMonitoringMode():
		return oa.HostMonitoring.Version
	}

	return ""
}

// CustomOneAgentImage provides the image reference for the OneAgent provided in the Spec.
func (oa *OneAgent) CustomOneAgentImage() string {
	switch {
	case oa.ClassicFullStackMode():
		return oa.ClassicFullStack.Image
	case oa.HostMonitoringMode():
		return oa.HostMonitoring.Image
	case oa.CloudNativeFullstackMode():
		return oa.CloudNativeFullStack.Image
	}

	return ""
}

// DefaultOneAgentImage provides the image reference for the OneAgent from tenant registry.
func (oa *OneAgent) DefaultOneAgentImage(version string) string {
	if oa.apiUrlHost == "" {
		return ""
	}

	truncatedVersion := dtversion.ToImageTag(version)
	tag := truncatedVersion

	if !strings.Contains(tag, api.RawTag) {
		tag += "-" + api.RawTag
	}

	return oa.apiUrlHost + DefaultOneAgentImageRegistrySubPath + ":" + tag
}

func (oa *OneAgent) GetHostGroup() string {
	if oa.HostGroup != "" {
		return oa.HostGroup
	}

	return oa.HostGroupAsParam()
}

func (oa *OneAgent) HostGroupAsParam() string {
	var hostGroup string

	var args []string

	switch {
	case oa.CloudNativeFullstackMode() && oa.CloudNativeFullStack.Args != nil:
		args = oa.CloudNativeFullStack.Args
	case oa.ClassicFullStackMode() && oa.ClassicFullStack.Args != nil:
		args = oa.ClassicFullStack.Args
	case oa.HostMonitoringMode() && oa.HostMonitoring.Args != nil:
		args = oa.HostMonitoring.Args
	}

	for _, arg := range args {
		key, value := splitArg(arg)
		if key == "--set-host-group" {
			hostGroup = value

			break
		}
	}

	return hostGroup
}

func splitArg(arg string) (key, value string) {
	split := strings.Split(arg, "=")

	const expectedLen = 2

	if len(split) != expectedLen {
		return
	}

	key = split[0]
	value = split[1]

	return
}

func (oa *OneAgent) IsOneAgentCommunicationRouteClear() bool {
	return len(oa.ConnectionInfoStatus.CommunicationHosts) > 0
}

func (oa *OneAgent) GetOneAgentEnvironment() []corev1.EnvVar {
	switch {
	case oa.CloudNativeFullstackMode():
		return oa.CloudNativeFullStack.Env
	case oa.ClassicFullStackMode():
		return oa.ClassicFullStack.Env
	case oa.HostMonitoringMode():
		return oa.HostMonitoring.Env
	}

	return []corev1.EnvVar{}
}

func (oa *OneAgent) OneAgentEndpoints() string {
	return oa.ConnectionInfoStatus.Endpoints
}

// CodeModulesVersion provides version set in Status for the CodeModules.
func (oa *OneAgent) CodeModulesVersion() string {
	return oa.CodeModulesStatus.Version
}

// CodeModulesImage provides the image reference set in Status for the CodeModules.
// Format: repo@sha256:digest.
func (oa *OneAgent) CodeModulesImage() string {
	return oa.CodeModulesStatus.ImageID
}
