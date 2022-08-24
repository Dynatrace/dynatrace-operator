package statefulset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
)

type ModifierMock struct {
	mock.Mock
}

func NewModifierMock() *ModifierMock {
	return &ModifierMock{}
}

func (m *ModifierMock) Modify(sts *appsv1.StatefulSet) {
	m.Called(sts)
}

func TestStatefulsetBuilder(t *testing.T) {
	t.Run("Simple, no modifiers", func(t *testing.T) {
		b := Builder{}
		actual := b.Build()
		expected := appsv1.StatefulSet{}
		assert.Equal(t, expected, actual)
	})
	t.Run("One modifier", func(t *testing.T) {
		b := Builder{}

		modifierMock := NewModifierMock()
		modifierMock.On("Modify", mock.Anything).Return()

		b.AddModifier(modifierMock)
		actual := b.Build()

		modifierMock.AssertNumberOfCalls(t, "Modify", 1)

		expected := appsv1.StatefulSet{}
		assert.Equal(t, expected, actual)
	})
	t.Run("Two modifiers, one used twice", func(t *testing.T) {
		b := Builder{}

		modifierMock0 := NewModifierMock()
		modifierMock0.On("Modify", mock.Anything).Return()
		modifierMock1 := NewModifierMock()
		modifierMock1.On("Modify", mock.Anything).Return()

		b.AddModifier(modifierMock0, modifierMock0, modifierMock1)

		modifierMock0.AssertNumberOfCalls(t, "Modify", 2)
		modifierMock1.AssertNumberOfCalls(t, "Modify", 1)

		actual := b.Build()
		expected := appsv1.StatefulSet{}
		assert.Equal(t, expected, actual)
	})
}
