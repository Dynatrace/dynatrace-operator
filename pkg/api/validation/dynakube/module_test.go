package validation

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/operatorconfig"
	"github.com/stretchr/testify/assert"
)

func TestIsModuleDisabled(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		title           string
		dk              dynakube.DynaKube
		modules         operatorconfig.Modules
		moduleFunc      validatorFunc
		expectedMessage string
	}

	testCases := []testCase{
		{
			title:           "oa module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}}},
			modules:         operatorconfig.Modules{OneAgent: false},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: errorOneAgentModuleDisabled,
		},
		{
			title:           "oa module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: nil}}},
			modules:         operatorconfig.Modules{OneAgent: false},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "oa module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}}},
			modules:         operatorconfig.Modules{OneAgent: true},
			moduleFunc:      isOneAgentModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ag module disabled but also configured in dk => error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: dynakube.ActiveGateSpec{Capabilities: []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName}}}},
			modules:         operatorconfig.Modules{ActiveGate: false},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: errorActiveGateModuleDisabled,
		},
		{
			title:           "ag module disabled but not configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{}},
			modules:         operatorconfig.Modules{ActiveGate: false},
			moduleFunc:      isActiveGateModuleDisabled,
			expectedMessage: "",
		},
		{
			title:           "ag module enabled and also configured => no error",
			dk:              dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ActiveGate: dynakube.ActiveGateSpec{Capabilities: []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName}}}},
			modules:         operatorconfig.Modules{ActiveGate: true},
			moduleFunc:      isActiveGateModuleDisabled,
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
