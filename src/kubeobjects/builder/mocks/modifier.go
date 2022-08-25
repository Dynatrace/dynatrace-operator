package mocks

import "github.com/stretchr/testify/mock"

type ModifierMock[T any] struct {
	mock.Mock
}

func NewModifierMock[T any]() *ModifierMock[T] {
	return &ModifierMock[T]{}
}

func (m *ModifierMock[T]) Modify(data *T) {
	m.Called(data)
}
