package builder

import (
	mocks2 "github.com/Dynatrace/dynatrace-operator/pkg/util/builder/mocks"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStatefulsetBuilder(t *testing.T) {
	t.Run("Simple, no modifiers", func(t *testing.T) {
		b := GenericBuilder[mocks2.DataMock]{}
		actual, err := b.Build()
		assert.NoError(t, err)
		expected := mocks2.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("One modifier", func(t *testing.T) {
		b := GenericBuilder[mocks2.DataMock]{}

		modifierMock := mocks2.NewModifierMock[mocks2.DataMock]()
		modifierMock.On("Modify", mock.Anything).Return(nil)
		modifierMock.On("Enabled").Return(true)

		actual, _ := b.AddModifier(modifierMock).Build()

		modifierMock.AssertNumberOfCalls(t, "Modify", 1)

		expected := mocks2.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("One modifier, not enabled", func(t *testing.T) {
		b := GenericBuilder[mocks2.DataMock]{}

		modifierMock := mocks2.NewModifierMock[mocks2.DataMock]()
		modifierMock.On("Modify", mock.Anything).Return(nil)
		modifierMock.On("Enabled").Return(false)

		actual, _ := b.AddModifier(modifierMock).Build()

		modifierMock.AssertNumberOfCalls(t, "Modify", 0)

		expected := mocks2.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("Two modifiers, one used twice", func(t *testing.T) {
		b := GenericBuilder[mocks2.DataMock]{}

		modifierMock0 := mocks2.NewModifierMock[mocks2.DataMock]()
		modifierMock0.On("Modify", mock.Anything).Return(nil)
		modifierMock0.On("Enabled").Return(true)
		modifierMock1 := mocks2.NewModifierMock[mocks2.DataMock]()
		modifierMock1.On("Modify", mock.Anything).Return(nil)
		modifierMock1.On("Enabled").Return(true)

		actual, _ := b.AddModifier(modifierMock0, modifierMock0, modifierMock1).Build()

		modifierMock0.AssertNumberOfCalls(t, "Modify", 2)
		modifierMock1.AssertNumberOfCalls(t, "Modify", 1)

		expected := mocks2.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("Chain of modifiers", func(t *testing.T) {
		b := GenericBuilder[mocks2.DataMock]{}

		modifierMock0 := mocks2.NewModifierMock[mocks2.DataMock]()
		modifierMock0.On("Modify", mock.Anything).Return(nil)
		modifierMock0.On("Enabled").Return(true)
		modifierMock1 := mocks2.NewModifierMock[mocks2.DataMock]()
		modifierMock1.On("Modify", mock.Anything).Return(nil)
		modifierMock1.On("Enabled").Return(true)

		actual, _ := b.AddModifier(modifierMock0, modifierMock0).AddModifier(modifierMock1).Build()

		modifierMock0.AssertNumberOfCalls(t, "Modify", 2)
		modifierMock1.AssertNumberOfCalls(t, "Modify", 1)

		expected := mocks2.DataMock{}
		assert.Equal(t, expected, actual)
	})
}
