package capability

import "github.com/stretchr/testify/mock"

type MockReconciler struct {
	mock.Mock
}

func (m *MockReconciler) Reconcile() error {
	args := m.Called()
	return args.Error(0)
}
