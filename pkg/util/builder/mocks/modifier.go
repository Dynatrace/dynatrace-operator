package mocks

import "github.com/stretchr/testify/mock"

type ModifierMock[T any] struct {
	mock.Mock
}

func NewModifierMock[T any]() *ModifierMock[T] {
	return &ModifierMock[T]{}
}

func (m *ModifierMock[T]) Enabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *ModifierMock[T]) Modify(data *T) error {
	args := m.Called(data)
	return args.Error(0)
}
