// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

func TestBuilder(t *testing.T) {
	t.Run("Simple, no modifiers", func(t *testing.T) {
		b := Builder{}
		actual, err := b.Build()
		require.NoError(t, err)

		expected := &appsv1.StatefulSet{}
		require.Equal(t, expected, actual)
	})
	t.Run("One modifier", func(t *testing.T) {
		b := Builder{}

		modifierMock := NewMockModifier(t)
		modifierMock.EXPECT().Modify(mock.Anything).Return(nil).Once()
		modifierMock.EXPECT().Enabled().Return(true).Once()

		actual, err := b.AddModifier(modifierMock).Build()
		require.NoError(t, err)

		expected := &appsv1.StatefulSet{}
		require.Equal(t, expected, actual)
	})
	t.Run("One modifier, not enabled", func(t *testing.T) {
		b := Builder{}

		modifierMock := NewMockModifier(t)
		modifierMock.EXPECT().Enabled().Return(false).Once()

		actual, err := b.AddModifier(modifierMock).Build()
		require.NoError(t, err)

		expected := &appsv1.StatefulSet{}
		require.Equal(t, expected, actual)
	})
	t.Run("Two modifiers, one used twice", func(t *testing.T) {
		b := Builder{}

		modifierMock0 := NewMockModifier(t)
		modifierMock0.EXPECT().Modify(mock.Anything).Return(nil).Twice()
		modifierMock0.EXPECT().Enabled().Return(true).Twice()

		modifierMock1 := NewMockModifier(t)
		modifierMock1.EXPECT().Modify(mock.Anything).Return(nil).Once()
		modifierMock1.EXPECT().Enabled().Return(true).Once()

		actual, err := b.AddModifier(modifierMock0, modifierMock0, modifierMock1).Build()
		require.NoError(t, err)

		expected := &appsv1.StatefulSet{}
		require.Equal(t, expected, actual)
	})
	t.Run("Chain of modifiers", func(t *testing.T) {
		b := Builder{}

		modifierMock0 := NewMockModifier(t)
		modifierMock0.EXPECT().Modify(mock.Anything).Return(nil).Twice()
		modifierMock0.EXPECT().Enabled().Return(true).Twice()

		modifierMock1 := NewMockModifier(t)
		modifierMock1.EXPECT().Modify(mock.Anything).Return(nil).Once()
		modifierMock1.EXPECT().Enabled().Return(true).Once()

		actual, err := b.AddModifier(modifierMock0, modifierMock0).AddModifier(modifierMock1).Build()
		require.NoError(t, err)

		expected := &appsv1.StatefulSet{}
		require.Equal(t, expected, actual)
	})
}
