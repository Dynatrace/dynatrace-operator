package activegate

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtversion"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type mockUpdateService struct {
	mock.Mock
}

func (mus *mockUpdateService) FindOutdatedPods(r *ReconcileActiveGate, logger logr.Logger, instance *dynatracev1alpha1.DynaKube) ([]corev1.Pod, error) {
	args := mus.Called(r, logger, instance)
	return args.Get(0).([]corev1.Pod), args.Error(1)
}

func (mus *mockUpdateService) IsLatest(validator dtversion.ReleaseValidator) (bool, error) {
	args := mus.Called(validator)
	return args.Get(0).(bool), args.Error(1)
}

func (mus *mockUpdateService) UpdatePods(r *ReconcileActiveGate, instance *dynatracev1alpha1.DynaKube) (*reconcile.Result, error) {
	args := mus.Called(r, instance)
	return args.Get(0).(*reconcile.Result), args.Error(1)
}
