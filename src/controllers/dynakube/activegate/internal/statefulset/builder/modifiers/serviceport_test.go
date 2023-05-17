package modifiers

import (
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setServicePortUsage(dynakube *dynatracev1.DynaKube, isUsed bool) {
	if isUsed {
		dynakube.Spec.ActiveGate.Capabilities = append(dynakube.Spec.ActiveGate.Capabilities, dynatracev1.MetricsIngestCapability.DisplayName)
	}
}

func TestServicePortEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)

		mod := NewServicePortModifier(dynakube, multiCapability)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, false)
		multiCapability := capability.NewMultiCapability(&dynakube)

		mod := NewServicePortModifier(dynakube, multiCapability)

		assert.False(t, mod.Enabled())
	})
}

func TestServicePortModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)
		mod := NewServicePortModifier(dynakube, multiCapability)
		builder := createBuilderForTesting()
		expectedPorts := mod.getPorts()
		expectedEnv := mod.getEnvs()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		container := sts.Spec.Template.Spec.Containers[0]
		isSubset(t, expectedPorts, container.Ports)
		isSubset(t, expectedEnv, container.Env)
		assert.Equal(t, consts.HttpsServicePortName, container.ReadinessProbe.HTTPGet.Port.StrVal)
	})
}

func TestBuildServiceNameForDNSEntryPoint(t *testing.T) {
	actual := buildServiceHostName("test-name", "test-component-feature")
	assert.NotEmpty(t, actual)

	expected := "$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_HOST):$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_PORT)"
	assert.Equal(t, expected, actual)

	testStringName := "this---test_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_HOST):$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_PORT)"
	actual = buildServiceHostName(testStringName, testStringFeature)
	assert.Equal(t, expected, actual)
}
