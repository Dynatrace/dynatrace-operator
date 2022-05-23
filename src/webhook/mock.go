package webhook

import (
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
)

type PodMutatorMock struct {
	mock.Mock
}

var _ PodMutator = &PodMutatorMock{}

func (mutator *PodMutatorMock) Enabled(pod *corev1.Pod) bool {
	args := mutator.Called(pod)
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
