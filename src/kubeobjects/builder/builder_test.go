package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DataMock struct {
	mock.Mock
}

type ModifierMock[T any] struct {
	mock.Mock
}

func NewModifierMock[T any]() *ModifierMock[T] {
	return &ModifierMock[T]{}
}

func (m *ModifierMock[T]) Modify(data *T) {
	m.Called(data)
}

func TestStatefulsetBuilder(t *testing.T) {
	t.Run("Simple, no modifiers", func(t *testing.T) {
		b := Builder[DataMock]{}
		actual := b.Build()
		expected := DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("One modifier", func(t *testing.T) {
		b := Builder[DataMock]{}

		modifierMock := NewModifierMock[DataMock]()
		modifierMock.On("Modify", mock.Anything).Return()

		b.AddModifier(modifierMock)
		actual := b.Build()

		modifierMock.AssertNumberOfCalls(t, "Modify", 1)

		expected := DataMock{}
		assert.Equal(t, expected, actual)
	})
	t.Run("Two modifiers, one used twice", func(t *testing.T) {
		b := Builder[DataMock]{}

		modifierMock0 := NewModifierMock[DataMock]()
		modifierMock0.On("Modify", mock.Anything).Return()
		modifierMock1 := NewModifierMock[DataMock]()
		modifierMock1.On("Modify", mock.Anything).Return()

		b.AddModifier(modifierMock0, modifierMock0, modifierMock1)

		modifierMock0.AssertNumberOfCalls(t, "Modify", 2)
		modifierMock1.AssertNumberOfCalls(t, "Modify", 1)

		actual := b.Build()
		expected := DataMock{}
		assert.Equal(t, expected, actual)
	})
}
