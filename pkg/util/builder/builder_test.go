package builder

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/builder/mocks"
	modifiermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/util/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	t.Run("Simple, no modifiers", func(t *testing.T) {
		b := GenericBuilder[mocks.DataMock]{}
		actual, err := b.Build()
		require.NoError(t, err)

		expected := mocks.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("One modifier", func(t *testing.T) {
		b := GenericBuilder[mocks.DataMock]{}

		modifierMock := modifiermock.NewModifier[mocks.DataMock](t)
		modifierMock.On("Modify", mock.Anything).Return(nil)
		modifierMock.On("Enabled").Return(true)

		actual, err := b.AddModifier(modifierMock).Build()
		require.NoError(t, err)

		modifierMock.AssertNumberOfCalls(t, "Modify", 1)

		expected := mocks.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("One modifier, not enabled", func(t *testing.T) {
		b := GenericBuilder[mocks.DataMock]{}

		modifierMock := modifiermock.NewModifier[mocks.DataMock](t)
		modifierMock.On("Modify", mock.Anything).Return(nil).Maybe()
		modifierMock.On("Enabled").Return(false)

		actual, err := b.AddModifier(modifierMock).Build()
		require.NoError(t, err)

		modifierMock.AssertNumberOfCalls(t, "Modify", 0)

		expected := mocks.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("Two modifiers, one used twice", func(t *testing.T) {
		b := GenericBuilder[mocks.DataMock]{}

		modifierMock0 := modifiermock.NewModifier[mocks.DataMock](t)
		modifierMock0.On("Modify", mock.Anything).Return(nil)
		modifierMock0.On("Enabled").Return(true)

		modifierMock1 := modifiermock.NewModifier[mocks.DataMock](t)
		modifierMock1.On("Modify", mock.Anything).Return(nil)
		modifierMock1.On("Enabled").Return(true)

		actual, err := b.AddModifier(modifierMock0, modifierMock0, modifierMock1).Build()
		require.NoError(t, err)

		modifierMock0.AssertNumberOfCalls(t, "Modify", 2)
		modifierMock1.AssertNumberOfCalls(t, "Modify", 1)

		expected := mocks.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("Chain of modifiers", func(t *testing.T) {
		b := GenericBuilder[mocks.DataMock]{}

		modifierMock0 := modifiermock.NewModifier[mocks.DataMock](t)
		modifierMock0.On("Modify", mock.Anything).Return(nil)
		modifierMock0.On("Enabled").Return(true)

		modifierMock1 := modifiermock.NewModifier[mocks.DataMock](t)
		modifierMock1.On("Modify", mock.Anything).Return(nil)
		modifierMock1.On("Enabled").Return(true)

		actual, err := b.AddModifier(modifierMock0, modifierMock0).AddModifier(modifierMock1).Build()
		require.NoError(t, err)

		modifierMock0.AssertNumberOfCalls(t, "Modify", 2)
		modifierMock1.AssertNumberOfCalls(t, "Modify", 1)

		expected := mocks.DataMock{}
		assert.Equal(t, expected, actual)
	})
}
