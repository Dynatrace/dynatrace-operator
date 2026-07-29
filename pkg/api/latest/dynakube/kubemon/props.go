// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package kubemon

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	corev1 "k8s.io/api/core/v1"
)

const (
	KubeMonAvailableConditionType = "KubernetesMonitoringAvailable"

	NameSuffix = "-kubemon"

	ServiceAccountName = "dynatrace-activegate"

	TenantRegistrySubPath = "/linux/activegate"
)

// KubeMon wraps Spec and Status for ergonomic access via dk.KubernetesMonitoring().
type KubeMon struct {
	*Spec
	*Status

	name       string
	apiURLHost string
}

func (km *Spec) IsEnabled() bool {
	return km != nil
}

// SetName seeds the DynaKube name onto the wrapper.
func (km *KubeMon) SetName(name string) {
	km.name = name
}

func (km *KubeMon) SetAPIURLHost(apiURLHost string) {
	km.apiURLHost = apiURLHost
}

func (km *Spec) GetServiceAccountName() string {
	return ServiceAccountName
}

func (km *KubeMon) GetStatefulSetName() string {
	return km.name + NameSuffix
}

func (km *KubeMon) GetConnectionInfoConfigMapName() string {
	return km.name + NameSuffix + "-connection-info"
}

func (km *KubeMon) GetTenantSecretName() string {
	return km.name + NameSuffix + "-tenant-secret"
}

func (km *KubeMon) GetAuthTokenSecretName() string {
	return km.name + NameSuffix + "-authtoken-secret"
}

func (km *Spec) GetPullPolicy() corev1.PullPolicy {
	if km == nil {
		return ""
	}

	return corev1.PullPolicy(km.ImagePullPolicy)
}

// GetCustomImage returns the user-provided image override, or "" if unset.
func (km *Spec) GetCustomImage() string {
	if km == nil {
		return ""
	}

	return km.Image
}

func (km *KubeMon) GetDefaultImage(version string) string {
	if km.apiURLHost == "" {
		return ""
	}

	truncatedVersion := dtversion.ToImageTag(version)
	tag := truncatedVersion

	if !strings.HasSuffix(tag, api.RawTag) {
		tag += "-" + api.RawTag
	}

	return km.apiURLHost + TenantRegistrySubPath + ":" + tag
}
