package validation

import (
	"context"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/stretchr/testify/assert"
)

func TestIsModuleDisabled(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		title           string
		dk              dynakube.DynaKube
		modules         installconfig.Modules
		moduleFunc      validatorFunc
		expectedMessage string
	}

	testCases := []testCase{
		{
			title:           "oa module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{}}}},
			modules:         installconfig.Modules{OneAgent: false, CSIDriver: true},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: errorOneAgentModuleDisabled,
		},
		{
			title:           "oa module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{CloudNativeFullStack: nil}}},
			modules:         installconfig.Modules{OneAgent: false, CSIDriver: true},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "oa module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{}}}},
			modules:         installconfig.Modules{OneAgent: true, CSIDriver: true},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ecc module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}}}},
			modules:         installconfig.Modules{Extensions: false},
			moduleFunc:      isExtensionsModuleDisabled,
			expectedMessage: errorExtensionsModuleDisabled,
		},
		{
			title:           "ecc module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{}},
			modules:         installconfig.Modules{Extensions: false},
			moduleFunc:      isExtensionsModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ecc module disabled but prometheus extension enabled => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}}}},
			modules:         installconfig.Modules{Extensions: false},
			moduleFunc:      isExtensionsModuleDisabled,
			expectedMessage: errorExtensionsModuleDisabled,
		},
		{
			title:           "ecc module enabled and prometheus extension enabled => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}}}},
			modules:         installconfig.Modules{Extensions: true},
			moduleFunc:      isExtensionsModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ecc module disabled but databases extension enabled => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Extensions: &extensions.Spec{Databases: []extensions.DatabaseSpec{{ID: "test"}}}}},
			modules:         installconfig.Modules{Extensions: false},
			moduleFunc:      isExtensionsModuleDisabled,
			expectedMessage: errorExtensionsModuleDisabled,
		},
		{
			title:           "ecc module enabled and databases extension enabled => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Extensions: &extensions.Spec{Databases: []extensions.DatabaseSpec{{ID: "test"}}}}},
			modules:         installconfig.Modules{Extensions: true},
			moduleFunc:      isExtensionsModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "logmonitoring module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{LogMonitoring: &logmonitoring.Spec{}}},
			modules:         installconfig.Modules{LogMonitoring: false},
			moduleFunc:      isLogMonitoringModuleDisabled,
			expectedMessage: errorLogMonitoringModuleDisabled,
		},
		{
			title:           "logmonitoring module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{}},
			modules:         installconfig.Modules{LogMonitoring: false},
			moduleFunc:      isLogMonitoringModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "logmonitoring module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{LogMonitoring: &logmonitoring.Spec{}}},
			modules:         installconfig.Modules{LogMonitoring: true},
			moduleFunc:      isLogMonitoringModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "kspm module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Kspm: &kspm.Spec{}}},
			modules:         installconfig.Modules{KSPM: false, KubernetesMonitoring: true},
			moduleFunc:      isKSPMDisabled,
			expectedMessage: errorKSPMModuleDisabled,
		},
		{
			title:           "kspm module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{}},
			modules:         installconfig.Modules{KSPM: false, KubernetesMonitoring: true},
			moduleFunc:      isKSPMDisabled,
			expectedMessage: "",
		},
		{
			title:           "kspm module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Kspm: &kspm.Spec{}}},
			modules:         installconfig.Modules{KSPM: true, KubernetesMonitoring: true},
			moduleFunc:      isKSPMDisabled,
			expectedMessage: "",
		},
		{
			title:           "kspm module enabled and also configured, no kubemon module enabled => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Kspm: &kspm.Spec{}}},
			modules:         installconfig.Modules{KSPM: true, KubernetesMonitoring: false},
			moduleFunc:      isKSPMDisabled,
			expectedMessage: errorKSPMDependsOnKubernetesMonitoringModule,
		},
		{
			title:           "kspm module disabled and also configured, no kubemon module enabled => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Kspm: &kspm.Spec{}}},
			modules:         installconfig.Modules{KSPM: false, KubernetesMonitoring: false},
			moduleFunc:      isKSPMDisabled,
			expectedMessage: strings.Join([]string{errorKSPMModuleDisabled, errorKSPMDependsOnKubernetesMonitoringModule}, ","),
		},
		{
			title:           "ag module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName}}}},
			modules:         installconfig.Modules{ActiveGate: false},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: errorActiveGateModuleDisabled,
		},
		{
			title:           "ag module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{}},
			modules:         installconfig.Modules{ActiveGate: false},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ag module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName}}}},
			modules:         installconfig.Modules{ActiveGate: true},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "dk has kubemon configured, available rbac [ag, kspm] => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}}}},
			modules:         installconfig.Modules{ActiveGate: true, KSPM: true, KubernetesMonitoring: false},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: errorKubernetesMonitoringModuleDisabled,
		},
		{
			title:           "dk has kubemon configured, available rbac [ag, kspm, kubemon] => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}}}},
			modules:         installconfig.Modules{ActiveGate: true, KSPM: true, KubernetesMonitoring: true},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "dk has kubemon configured, available rbac [ag, kubemon] => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}}}},
			modules:         installconfig.Modules{ActiveGate: true, KubernetesMonitoring: true},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "dk has kubemon configured, available rbac [ag] => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}}}},
			modules:         installconfig.Modules{ActiveGate: true, KubernetesMonitoring: false},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: errorKubernetesMonitoringModuleDisabled,
		},
		{
			title:           "dk has kubemon configured, available rbac [kubemon] => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}}}},
			modules:         installconfig.Modules{ActiveGate: false, KubernetesMonitoring: true},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: errorActiveGateModuleDisabled,
		},
	}

	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			errMsg := test.moduleFunc(ctx, &Validator{modules: test.modules}, &test.dk)
			assert.Equal(t, test.expectedMessage, errMsg)
		})
	}
}
