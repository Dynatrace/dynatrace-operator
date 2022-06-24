package manager

import (
	"context"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type Mock struct {
	TestManager
	mock.Mock
}

func (mgr *Mock) Start(ctx context.Context) error {
	args := mgr.Called(ctx)
	return args.Error(0)
}

func (mgr *Mock) AddHealthzCheck(name string, check healthz.Checker) error {
	args := mgr.Called(name, check)
	return args.Error(0)
}

func (mgr *Mock) AddReadyzCheck(name string, check healthz.Checker) error {
	args := mgr.Called(name, check)
	return args.Error(0)
}
