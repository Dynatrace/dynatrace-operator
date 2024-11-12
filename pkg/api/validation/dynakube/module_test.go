package validation

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
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
			title:           "csi module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}}},
			modules:         installconfig.Modules{OneAgent: true, CSIDriver: false},
			moduleFunc:      isCSIModuleDisabled,
			expectedMessage: errorCSIModuleDisabled,
		},
		{
			title:           "csi module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: nil}}},
			modules:         installconfig.Modules{OneAgent: true, CSIDriver: false},
			moduleFunc:      isCSIModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "csi module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}}},
			modules:         installconfig.Modules{OneAgent: true, CSIDriver: true},
			moduleFunc:      isCSIModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "csi module disabled and app-monitoring configured => no error, as it's optional for app-monitoring",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{}}}},
			modules:         installconfig.Modules{OneAgent: true, CSIDriver: false},
			moduleFunc:      isCSIModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "csi module disabled and host-monitoring configured => no error, as it's optional for host-monitoring",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{HostMonitoring: &dynakube.HostInjectSpec{}}}},
			modules:         installconfig.Modules{OneAgent: true, CSIDriver: true},
			moduleFunc:      isCSIModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "oa module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}}},
			modules:         installconfig.Modules{OneAgent: false, CSIDriver: true},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: errorOneAgentModuleDisabled,
		},
		{
			title:           "oa module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: nil}}},
			modules:         installconfig.Modules{OneAgent: false, CSIDriver: true},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "oa module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}}},
			modules:         installconfig.Modules{OneAgent: true, CSIDriver: true},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ag module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}}}},
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
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}}}},
			modules:         installconfig.Modules{ActiveGate: true},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ecc module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Extensions: &dynakube.ExtensionsSpec{}}},
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
			title:           "ecc module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Extensions: &dynakube.ExtensionsSpec{}}},
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
			modules:         installconfig.Modules{KSPM: false},
			moduleFunc:      isKSPMDisabled,
			expectedMessage: errorKSPMDisabled,
		},
		{
			title:           "kspm module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{}},
			modules:         installconfig.Modules{KSPM: false},
			moduleFunc:      isKSPMDisabled,
			expectedMessage: "",
		},
		{
			title:           "kspm module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Kspm: &kspm.Spec{}}},
			modules:         installconfig.Modules{KSPM: true},
			moduleFunc:      isKSPMDisabled,
			expectedMessage: "",
		},
	}

	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			errMsg := test.moduleFunc(ctx, &Validator{modules: test.modules}, &test.dk)
			assert.Equal(t, test.expectedMessage, errMsg)
		})
	}
}

func TestIsCSIDriverRequired(t *testing.T) {
	t.Run("DynaKube with cloud native", func(t *testing.T) {
		dk := dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}}}
		assert.True(t, isCSIRequired(&dk))
	})

	t.Run("DynaKube with host monitoring", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}
		assert.False(t, isCSIRequired(&dk))
	})

	t.Run("DynaKube with application monitoring", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
				},
			},
		}
		assert.False(t, isCSIRequired(&dk))
	})
}
