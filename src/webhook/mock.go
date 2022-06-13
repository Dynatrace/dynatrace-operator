package webhook

import (
	"github.com/stretchr/testify/mock"
)

type PodMutatorMock struct {
	mock.Mock
}

var _ PodMutator = &PodMutatorMock{}

func (mutator *PodMutatorMock) Enabled(request *BaseRequest) bool {
	args := mutator.Called(request)
	return args.Bool(0)
}

func (mutator *PodMutatorMock) Injected(request *BaseRequest) bool {
	args := mutator.Called(request)
	return args.Bool(0)
}

func (mutator *PodMutatorMock) Mutate(request *MutationRequest) error {
	args := mutator.Called(request)
	return args.Error(0)
}
func (mutator *PodMutatorMock) Reinvoke(request *ReinvocationRequest) bool {
	args := mutator.Called(request)
	return args.Bool(0)
}
