package capability

import (
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

type MockReconciler struct {
	mock.Mock
}

func (m *MockReconciler) Reconcile(_ context.Context) error {
	args := m.Called()
	return args.Error(0)
}
