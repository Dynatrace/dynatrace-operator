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

// IsApplicationMonitoringMode returns true when application only section is used.
func (oa *OneAgent) IsApplicationMonitoringMode() bool {
	return oa.ApplicationMonitoring != nil
}

// IsCloudNativeFullstackMode returns true when cloud native fullstack section is used.
func (oa *OneAgent) IsCloudNativeFullstackMode() bool {
	return oa.CloudNativeFullStack != nil
}

// IsHostMonitoringMode returns true when host monitoring section is used.
func (oa *OneAgent) IsHostMonitoringMode() bool {
	return oa.HostMonitoring != nil
}

// IsClassicFullStackMode returns true when classic fullstack section is used.
func (oa *OneAgent) IsClassicFullStackMode() bool {
	return oa.ClassicFullStack != nil
}

// IsDaemonsetRequired returns true when a feature requires OneAgent instances.
func (oa *OneAgent) IsDaemonsetRequired() bool {
	return oa.IsClassicFullStackMode() || oa.IsCloudNativeFullstackMode() || oa.IsHostMonitoringMode()
}

func (oa *OneAgent) GetDaemonsetName() string {
	return fmt.Sprintf("%s-%s", oa.name, PodNameOsAgent)
}

func (oa *OneAgent) IsPrivilegedNeeded() bool {
	return oa.featureOneAgentPrivileged
}

func (oa *OneAgent) IsReadinessProbeNeeded() bool {
	return oa.Healthcheck != nil
}

func (oa *OneAgent) IsLivenessProbeNeeded() bool {
	return oa.Healthcheck != nil && !oa.featureOneAgentSkipLivenessProbe
}

// IsAutoUpdateEnabled returns true if the Operator should update OneAgent instances automatically.
func (oa *OneAgent) IsAutoUpdateEnabled() bool {
	switch {
	case oa.IsCloudNativeFullstackMode():
		return oa.CloudNativeFullStack.AutoUpdate == nil || *oa.CloudNativeFullStack.AutoUpdate
	case oa.IsHostMonitoringMode():
		return oa.HostMonitoring.AutoUpdate == nil || *oa.HostMonitoring.AutoUpdate
	case oa.IsClassicFullStackMode():
		return oa.ClassicFullStack.AutoUpdate == nil || *oa.ClassicFullStack.AutoUpdate
	default:
		return false
	}
}

// GetTenantSecret returns the name of the secret containing the token for the OneAgent.
func (oa *OneAgent) GetTenantSecret() string {
	return oa.name + OneAgentTenantSecretSuffix
}

func (oa *OneAgent) GetConnectionInfoConfigMapName() string {
	return oa.name + OneAgentConnectionInfoConfigMapSuffix
}

func (oa *OneAgent) IsReadOnlyOneAgentsMode() bool {
	return oa.IsCloudNativeFullstackMode() || (oa.IsHostMonitoringMode() && oa.IsCSIAvailable())
}

func (oa *OneAgent) IsAppInjectionNeeded() bool {
	return oa.IsCloudNativeFullstackMode() || oa.IsApplicationMonitoringMode()
}

func (oa *OneAgent) GetInitResources() *corev1.ResourceRequirements {
	if oa.IsApplicationMonitoringMode() {
		return oa.ApplicationMonitoring.InitResources
	} else if oa.IsCloudNativeFullstackMode() {
		return oa.CloudNativeFullStack.InitResources
	}

	return nil
}

func (oa *OneAgent) GetNamespaceSelector() *metav1.LabelSelector {
	switch {
	case oa.IsCloudNativeFullstackMode():
		return &oa.CloudNativeFullStack.NamespaceSelector
	case oa.IsApplicationMonitoringMode():
		return &oa.ApplicationMonitoring.NamespaceSelector
	}

	return nil
}

func (oa *OneAgent) GetSecCompProfile() string {
	switch {
	case oa.IsCloudNativeFullstackMode():
		return oa.CloudNativeFullStack.SecCompProfile
	case oa.IsHostMonitoringMode():
		return oa.HostMonitoring.SecCompProfile
	case oa.IsClassicFullStackMode():
		return oa.ClassicFullStack.SecCompProfile
	default:
		return ""
	}
}

func (oa *OneAgent) GetNodeSelector(fallbackNodeSelector map[string]string) map[string]string {
	switch {
	case oa.IsClassicFullStackMode():
		return oa.ClassicFullStack.NodeSelector
	case oa.IsHostMonitoringMode():
		return oa.HostMonitoring.NodeSelector
	case oa.IsCloudNativeFullstackMode():
		return oa.CloudNativeFullStack.NodeSelector
	}

	return fallbackNodeSelector
}

// GetImage provides the image reference set in Status for the OneAgent.
// Format: repo@sha256:digest.
func (oa *OneAgent) GetImage() string {
	return oa.Status.ImageID
}

// GetVersion provides version set in Status for the OneAgent.
func (oa *OneAgent) GetVersion() string {
	return oa.Status.Version
}

// GetCustomVersion provides the version for the OneAgent provided in the Spec.
func (oa *OneAgent) GetCustomVersion() string {
	switch {
	case oa.IsClassicFullStackMode():
		return oa.ClassicFullStack.Version
	case oa.IsCloudNativeFullstackMode():
		return oa.CloudNativeFullStack.Version
	case oa.IsApplicationMonitoringMode():
		return oa.ApplicationMonitoring.Version
	case oa.IsHostMonitoringMode():
		return oa.HostMonitoring.Version
	}

	return ""
}

// GetCustomImage provides the image reference for the OneAgent provided in the Spec.
func (oa *OneAgent) GetCustomImage() string {
	switch {
	case oa.IsClassicFullStackMode():
		return oa.ClassicFullStack.Image
	case oa.IsHostMonitoringMode():
		return oa.HostMonitoring.Image
	case oa.IsCloudNativeFullstackMode():
		return oa.CloudNativeFullStack.Image
	}

	return ""
}

// GetDefaultImage provides the image reference for the OneAgent from tenant registry.
func (oa *OneAgent) GetDefaultImage(version string) string {
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

	return oa.GetHostGroupAsParam()
}

func (oa *OneAgent) GetHostGroupAsParam() string {
	var hostGroup string

	var args []string

	switch {
	case oa.IsCloudNativeFullstackMode() && oa.CloudNativeFullStack.Args != nil:
		args = oa.CloudNativeFullStack.Args
	case oa.IsClassicFullStackMode() && oa.ClassicFullStack.Args != nil:
		args = oa.ClassicFullStack.Args
	case oa.IsHostMonitoringMode() && oa.HostMonitoring.Args != nil:
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

func (oa *OneAgent) IsCommunicationRouteClear() bool {
	return len(oa.ConnectionInfoStatus.CommunicationHosts) > 0
}

func (oa *OneAgent) GetEnvironment() []corev1.EnvVar {
	switch {
	case oa.IsCloudNativeFullstackMode():
		return oa.CloudNativeFullStack.Env
	case oa.IsClassicFullStackMode():
		return oa.ClassicFullStack.Env
	case oa.IsHostMonitoringMode():
		return oa.HostMonitoring.Env
	}

	return []corev1.EnvVar{}
}

func (oa *OneAgent) GetEndpoints() string {
	return oa.ConnectionInfoStatus.Endpoints
}

// GetCustomCodeModulesImage provides the image reference for the CodeModules provided in the Spec.
func (oa *OneAgent) GetCustomCodeModulesImage() string {
	if oa.IsCloudNativeFullstackMode() {
		return oa.CloudNativeFullStack.CodeModulesImage
	} else if oa.IsApplicationMonitoringMode() && oa.IsCSIAvailable() {
		return oa.ApplicationMonitoring.CodeModulesImage
	}

	return ""
}

// GetCustomCodeModulesVersion provides the version for the CodeModules provided in the Spec.
func (oa *OneAgent) GetCustomCodeModulesVersion() string {
	return oa.GetCustomVersion()
}

// GetCodeModulesVersion provides version set in Status for the CodeModules.
func (oa *OneAgent) GetCodeModulesVersion() string {
	return oa.CodeModulesStatus.Version
}

// GetCodeModulesImage provides the image reference set in Status for the CodeModules.
// Format: repo@sha256:digest.
func (oa *OneAgent) GetCodeModulesImage() string {
	return oa.CodeModulesStatus.ImageID
}
