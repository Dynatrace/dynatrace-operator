package dynakube

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OneAgentTenantSecretSuffix            = "-oneagent-tenant-secret"
	OneAgentConnectionInfoConfigMapSuffix = "-oneagent-connection-info"
	PodNameOsAgent                        = "oneagent"
	DefaultOneAgentImageRegistrySubPath   = "/linux/oneagent"
)

// ApplicationMonitoringMode returns true when application only section is used.
func (dk *DynaKube) ApplicationMonitoringMode() bool {
	return dk.Spec.OneAgent != OneAgentSpec{} && dk.Spec.OneAgent.ApplicationMonitoring != nil
}

// CloudNativeFullstackMode returns true when cloud native fullstack section is used.
func (dk *DynaKube) CloudNativeFullstackMode() bool {
	return dk.Spec.OneAgent != OneAgentSpec{} && dk.Spec.OneAgent.CloudNativeFullStack != nil
}

// HostMonitoringMode returns true when host monitoring section is used.
func (dk *DynaKube) HostMonitoringMode() bool {
	return dk.Spec.OneAgent != OneAgentSpec{} && dk.Spec.OneAgent.HostMonitoring != nil
}

// ClassicFullStackMode returns true when classic fullstack section is used.
func (dk *DynaKube) ClassicFullStackMode() bool {
	return dk.Spec.OneAgent != OneAgentSpec{} && dk.Spec.OneAgent.ClassicFullStack != nil
}

// NeedsOneAgent returns true when a feature requires OneAgent instances.
func (dk *DynaKube) NeedsOneAgent() bool {
	return dk.ClassicFullStackMode() || dk.CloudNativeFullstackMode() || dk.HostMonitoringMode()
}

func (dk *DynaKube) OneAgentDaemonsetName() string {
	return fmt.Sprintf("%s-%s", dk.Name, PodNameOsAgent)
}

func (dk *DynaKube) NeedsOneAgentPrivileged() bool {
	return dk.FeatureOneAgentPrivileged()
}

func (dk *DynaKube) NeedsOneAgentProbe() bool {
	return dk.Status.OneAgent.Healthcheck != nil
}

// ShouldAutoUpdateOneAgent returns true if the Operator should update OneAgent instances automatically.
func (dk *DynaKube) ShouldAutoUpdateOneAgent() bool {
	switch {
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.AutoUpdate
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.AutoUpdate
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.AutoUpdate
	default:
		return false
	}
}

// OneagentTenantSecret returns the name of the secret containing the token for the OneAgent.
func (dk *DynaKube) OneagentTenantSecret() string {
	return dk.Name + OneAgentTenantSecretSuffix
}

func (dk *DynaKube) OneAgentConnectionInfoConfigMapName() string {
	return dk.Name + OneAgentConnectionInfoConfigMapSuffix
}

func (dk *DynaKube) NeedsReadOnlyOneAgents() bool {
	return (dk.HostMonitoringMode() || dk.CloudNativeFullstackMode()) &&
		dk.FeatureReadOnlyOneAgent()
}

func (dk *DynaKube) NeedsCSIDriver() bool {
	isAppMonitoringWithCSI := dk.ApplicationMonitoringMode() &&
		dk.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver

	isHostMonitoringWithCSI := dk.HostMonitoringMode() && dk.FeatureReadOnlyOneAgent()

	return dk.CloudNativeFullstackMode() || isAppMonitoringWithCSI || isHostMonitoringWithCSI
}

func (dk *DynaKube) NeedAppInjection() bool {
	return dk.CloudNativeFullstackMode() || dk.ApplicationMonitoringMode()
}

func (dk *DynaKube) InitResources() *corev1.ResourceRequirements {
	if dk.ApplicationMonitoringMode() {
		return dk.Spec.OneAgent.ApplicationMonitoring.InitResources
	} else if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.InitResources
	}

	return nil
}

func (dk *DynaKube) OneAgentNamespaceSelector() *metav1.LabelSelector {
	switch {
	case dk.CloudNativeFullstackMode():
		return &dk.Spec.OneAgent.CloudNativeFullStack.NamespaceSelector
	case dk.ApplicationMonitoringMode():
		return &dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector
	}

	return nil
}

func (dk *DynaKube) OneAgentSecCompProfile() string {
	switch {
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.SecCompProfile
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.SecCompProfile
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.SecCompProfile
	default:
		return ""
	}
}

func (dk *DynaKube) OneAgentNodeSelector() map[string]string {
	switch {
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.NodeSelector
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.NodeSelector
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.NodeSelector
	}

	return nil
}

// CodeModulesVersion provides version set in Status for the CodeModules.
func (dk *DynaKube) CodeModulesVersion() string {
	return dk.Status.CodeModules.Version
}

// CodeModulesImage provides the image reference set in Status for the CodeModules.
// Format: repo@sha256:digest.
func (dk *DynaKube) CodeModulesImage() string {
	return dk.Status.CodeModules.ImageID
}

// CustomCodeModulesImage provides the image reference for the CodeModules provided in the Spec.
func (dk *DynaKube) CustomCodeModulesImage() string {
	if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.CodeModulesImage
	} else if dk.ApplicationMonitoringMode() && dk.NeedsCSIDriver() {
		return dk.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage
	}

	return ""
}

// CustomCodeModulesVersion provides the version for the CodeModules provided in the Spec.
func (dk *DynaKube) CustomCodeModulesVersion() string {
	if !dk.ApplicationMonitoringMode() {
		return ""
	}

	return dk.CustomOneAgentVersion()
}

// OneAgentImage provides the image reference set in Status for the OneAgent.
// Format: repo@sha256:digest.
func (dk *DynaKube) OneAgentImage() string {
	return dk.Status.OneAgent.ImageID
}

// OneAgentVersion provides version set in Status for the OneAgent.
func (dk *DynaKube) OneAgentVersion() string {
	return dk.Status.OneAgent.Version
}

// CustomOneAgentVersion provides the version for the OneAgent provided in the Spec.
func (dk *DynaKube) CustomOneAgentVersion() string {
	switch {
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.Version
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.Version
	case dk.ApplicationMonitoringMode():
		return dk.Spec.OneAgent.ApplicationMonitoring.Version
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.Version
	}

	return ""
}

// CustomOneAgentImage provides the image reference for the OneAgent provided in the Spec.
func (dk *DynaKube) CustomOneAgentImage() string {
	switch {
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.Image
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.Image
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.Image
	}

	return ""
}

// DefaultOneAgentImage provides the image reference for the OneAgent from tenant registry.
func (dk *DynaKube) DefaultOneAgentImage(version string) string {
	apiUrlHost := dk.ApiUrlHost()
	if apiUrlHost == "" {
		return ""
	}

	truncatedVersion := dtversion.ToImageTag(version)
	tag := truncatedVersion

	if !strings.Contains(tag, api.RawTag) {
		tag += "-" + api.RawTag
	}

	return apiUrlHost + DefaultOneAgentImageRegistrySubPath + ":" + tag
}

func (dk *DynaKube) HostGroup() string {
	if dk.Spec.OneAgent.HostGroup != "" {
		return dk.Spec.OneAgent.HostGroup
	}

	return dk.HostGroupAsParam()
}

func (dk *DynaKube) HostGroupAsParam() string {
	var hostGroup string

	var args []string

	switch {
	case dk.CloudNativeFullstackMode() && dk.Spec.OneAgent.CloudNativeFullStack.Args != nil:
		args = dk.Spec.OneAgent.CloudNativeFullStack.Args
	case dk.ClassicFullStackMode() && dk.Spec.OneAgent.ClassicFullStack.Args != nil:
		args = dk.Spec.OneAgent.ClassicFullStack.Args
	case dk.HostMonitoringMode() && dk.Spec.OneAgent.HostMonitoring.Args != nil:
		args = dk.Spec.OneAgent.HostMonitoring.Args
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

func (dk *DynaKube) IsOneAgentCommunicationRouteClear() bool {
	return len(dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts) > 0
}

func (dk *DynaKube) GetOneAgentEnvironment() []corev1.EnvVar {
	switch {
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.Env
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.Env
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.Env
	}

	return []corev1.EnvVar{}
}

func (dk *DynaKube) OneAgentEndpoints() string {
	return dk.Status.OneAgent.ConnectionInfoStatus.Endpoints
}
