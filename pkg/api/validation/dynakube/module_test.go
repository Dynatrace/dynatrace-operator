package validation

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}}},
			modules:         installconfig.Modules{OneAgent: false},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: errorOneAgentModuleDisabled,
		},
		{
			title:           "oa module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: nil}}},
			modules:         installconfig.Modules{OneAgent: false},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "oa module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}}},
			modules:         installconfig.Modules{OneAgent: true},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ag module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: dynakube.ActiveGateSpec{Capabilities: []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName}}}},
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
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: dynakube.ActiveGateSpec{Capabilities: []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName}}}},
			modules:         installconfig.Modules{ActiveGate: true},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ecc module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Extensions: dynakube.ExtensionsSpec{Enabled: true}}},
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
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Extensions: dynakube.ExtensionsSpec{Enabled: true}}},
			modules:         installconfig.Modules{Extensions: true},
			moduleFunc:      isExtensionsModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "logmodule module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{LogModule: dynakube.LogModuleSpec{Enabled: true}}},
			modules:         installconfig.Modules{LogModule: false},
			moduleFunc:      isLogModuleModuleDisabled,
			expectedMessage: errorLogModuleModuleDisabled,
		},
		{
			title:           "logmodule module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{}},
			modules:         installconfig.Modules{LogModule: false},
			moduleFunc:      isLogModuleModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "logmodule module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{LogModule: dynakube.LogModuleSpec{Enabled: true}}},
			modules:         installconfig.Modules{LogModule: true},
			moduleFunc:      isLogModuleModuleDisabled,
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
