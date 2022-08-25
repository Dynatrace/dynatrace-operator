package builder

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStatefulsetBuilder(t *testing.T) {
	t.Run("Simple, no modifiers", func(t *testing.T) {
		b := Builder[mocks.DataMock]{}
		actual := b.Build()
		expected := mocks.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("One modifier", func(t *testing.T) {
		b := Builder[mocks.DataMock]{}

		modifierMock := mocks.NewModifierMock[mocks.DataMock]()
		modifierMock.On("Modify", mock.Anything).Return()

		b.AddModifier(modifierMock)
		actual := b.Build()

		modifierMock.AssertNumberOfCalls(t, "Modify", 1)

		expected := mocks.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("Two modifiers, one used twice", func(t *testing.T) {
		b := Builder[mocks.DataMock]{}

		modifierMock0 := mocks.NewModifierMock[mocks.DataMock]()
		modifierMock0.On("Modify", mock.Anything).Return()
		modifierMock1 := mocks.NewModifierMock[mocks.DataMock]()
		modifierMock1.On("Modify", mock.Anything).Return()

		b.AddModifier(modifierMock0, modifierMock0, modifierMock1)
		actual := b.Build()

		modifierMock0.AssertNumberOfCalls(t, "Modify", 2)
		modifierMock1.AssertNumberOfCalls(t, "Modify", 1)

		expected := mocks.DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("Chain of modifiers", func(t *testing.T) {
		b := Builder[mocks.DataMock]{}

		modifierMock0 := mocks.NewModifierMock[mocks.DataMock]()
		modifierMock0.On("Modify", mock.Anything).Return()
		modifierMock1 := mocks.NewModifierMock[mocks.DataMock]()
		modifierMock1.On("Modify", mock.Anything).Return()

		b.AddModifier(modifierMock0, modifierMock0).AddModifier(modifierMock1)
		actual := b.Build()

		modifierMock0.AssertNumberOfCalls(t, "Modify", 2)
		modifierMock1.AssertNumberOfCalls(t, "Modify", 1)

		expected := mocks.DataMock{}
		assert.Equal(t, expected, actual)
	})

}
